package server

import (
	"context"
	"fmt"
	code "github.com/ethereum/go-ethereum/msequencer/errors/errorcode"
	"github.com/ethereum/go-ethereum/msequencer/server/processor"
	"github.com/ethereum/go-ethereum/msequencer/set"
	"io"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	codeErr "github.com/ethereum/go-ethereum/msequencer/errors"
	scommon "github.com/ethereum/go-ethereum/msequencer/server/common"
)

type HttpHandler struct {
	run       int32
	codecsMu  sync.Mutex
	codecs    *set.Set
	processor *processor.Processor
}

func NewHttpHandler() (*HttpHandler, error) {
	p, err := processor.NewProcessor()
	if err != nil {
		return nil, err
	}
	handler := &HttpHandler{
		codecs:    set.New(set.ThreadSafe).(*set.Set),
		run:       1,
		processor: p,
	}

	return handler, nil
}

// ServeCodec reads incoming requests from codec, calls the appropriate callback and writes the
// response back using the given codec. It will block until the codec is closed or the server is
// stopped. In either case the codec is closed.
func (s *HttpHandler) ServeCodec(ctx context.Context, codec ServerCodec, options CodecOption) error {
	defer codec.Close()
	return s.serveRequest(ctx, codec, false, options)
}

// ServeSingleRequest reads and processes a single RPC request from the given codec. It will not
// close the codec unless a non-recoverable error has occurred. Note, this method will return after
// a single request has been processed!
func (s *HttpHandler) ServeSingleRequest(codec ServerCodec, options CodecOption) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return s.serveRequest(ctx, codec, true, options)
}

// serveRequest will reads requests from the codec, calls the RPC callback and
// writes the response to the given codec.
//
// If singleShot is true it will process a single request, otherwise it will handle
// requests until the codec returns an error when reading a request (in most cases
// an EOF). It executes requests in parallel when singleShot is false.
func (s *HttpHandler) serveRequest(ctx context.Context, codec ServerCodec, singleShot bool, options CodecOption) error {

	var pend sync.WaitGroup

	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
		}

		s.codecsMu.Lock()
		s.codecs.Remove(codec)
		s.codecsMu.Unlock()

		return
	}()

	s.codecsMu.Lock()
	if atomic.LoadInt32(&s.run) != 1 { // server stopped
		s.codecsMu.Unlock()
		return &codeErr.ShutdownError{}
	}
	s.codecs.Add(codec)
	s.codecsMu.Unlock()

	// test if the server is ordered to stop
	for atomic.LoadInt32(&s.run) == 1 {
		reqs, batch, err := s.readRequest(codec, options)
		// If a parsing error occurred, send an error
		if err != nil {
			// If a parsing error occurred, send an error
			if err.Error() != "EOF" {
				codec.Write(codec.CreateErrorResponse(nil, err))
			}
			// Error or end of stream, wait for requests and tear down
			pend.Wait()
			return nil
		}

		// check if server is ordered to shutdown and return an error
		// telling the client that his request failed.
		if atomic.LoadInt32(&s.run) != 1 {
			err := &codeErr.ShutdownError{}
			if batch {
				resps := make([]interface{}, len(reqs))
				for i, r := range reqs {
					resps[i] = codec.CreateErrorResponse(r.ID, err)
				}
				return codec.Write(resps)
			}
			return codec.Write(codec.CreateErrorResponse(reqs[0].ID, err))
		}

		if singleShot {
			s.handleReqs(ctx, codec, reqs)
			return nil
		}

		// For multi-shot connections, start a goroutine to serve and loop back
		pend.Add(1)

		go func() {
			defer pend.Done()
			s.handleReqs(ctx, codec, reqs)
		}()

	}
	return nil
}

// Stop will stop reading new requests, wait for stopPendingRequestTimeout to
// allow pending requests to finish, close all codecs which will cancels pending
// requests/subscriptions.
func (s *HttpHandler) Stop() {
	if atomic.CompareAndSwapInt32(&s.run, 1, 0) {
		time.AfterFunc(stopPendingRequestTimeout, func() {
			s.codecsMu.Lock()
			defer s.codecsMu.Unlock()
			s.codecs.Each(func(c interface{}) bool {
				c.(ServerCodec).Close()
				return true
			})
		})
	}
}

// readRequest requests the next (batch) request from the codec. It will return the collection
// of requests, an indication if the request was a batch, the invalid request identifier and an
// error when the request could not be read/parsed.
func (s *HttpHandler) readRequest(codec ServerCodec, options CodecOption) ([]*scommon.RPCRequest, bool, code.RPCError) {
	reqs, batch, err := codec.ReadRawRequest(options)
	if err != nil {
		return nil, batch, err
	}

	if len(reqs) == 0 {
		return nil, false, &codeErr.InvalidRequestError{Message: "no request found"}
	}
	return reqs, batch, nil
}

// handleReqs will handle RPC request array and write result then send to client
func (s *HttpHandler) handleReqs(ctx context.Context, codec ServerCodec, reqs []*scommon.RPCRequest) {
	number := len(reqs)
	response := make([]interface{}, number)

	i := 0
	for _, req := range reqs {
		req.Ctx = ctx
		//TODO: whether can ignore http check
		//if err := codec.CheckHTTPHeaders(req.Namespace, req.Method); err != nil {
		//	logger.Errorf("CheckHTTPHeaders error: %v", err)
		//	response[i] = codec.CreateErrorResponse(req.ID, req.Namespace, &common.CertError{Message: err.Error()})
		//	break
		//}
		response[i] = s.handleChannelReq(codec, req)
		i++
	}

	if number == 1 {
		if err := codec.Write(response[0]); err != nil {
			codec.Close()
		}
	} else {
		if err := codec.Write(response); err != nil {
			codec.Close()
		}
	}
}

// handleChannelReq implements receiver.handleChannelReq interface to handle
// request in channel and return jsonrpc response.
func (s *HttpHandler) handleChannelReq(codec ServerCodec, req *scommon.RPCRequest) interface{} {
	// truly call the RPCRequest request processor(TODO: consider later)

	r := s.processor.ProcessRequest(req)
	if r == nil {
		return codec.CreateErrorResponse(req.ID, &codeErr.CallbackError{Message: "no process result"})
	}

	if response, ok := r.(*scommon.RPCResponse); ok {
		if response.Error != nil && response.Reply == nil {
			return codec.CreateErrorResponse(response.ID, response.Error)
		} else if response.Error != nil && response.Reply != nil {
			return codec.CreateErrorResponseWithInfo(response.ID, response.Error, response.Reply)
		} else if response.Reply != nil {
			return codec.CreateResponse(response.ID, response.Reply)

		} else {
			return codec.CreateResponse(response.ID, nil)
		}
	} else {
		return codec.CreateErrorResponse(req.ID, &codeErr.CallbackError{Message: "response type invalid!"})
	}
}

func (srv *HttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > maxHTTPRequestContentLength {
		http.Error(w,
			fmt.Sprintf("content length too large (%d>%d)", r.ContentLength, maxHTTPRequestContentLength),
			http.StatusRequestEntityTooLarge)
		return
	}

	w.Header().Set("content-type", "application/json")
	codec := NewJSONCodec(&httpReadWrite{r.Body, w}, r)
	defer codec.Close()
	srv.ServeSingleRequest(codec, OptionMethodInvocation)
}

type httpReadWrite struct {
	io.Reader
	io.Writer
}

func (hrw *httpReadWrite) Close() error { return nil }
