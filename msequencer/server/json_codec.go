package server

import (
	"encoding/json"
	"errors"
	"fmt"
	code "github.com/ethereum/go-ethereum/msequencer/errors/errorcode"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"

	codeErr "github.com/ethereum/go-ethereum/msequencer/errors"
	scommon "github.com/ethereum/go-ethereum/msequencer/server/common"
)

type jsonCodecImpl struct {
	closer sync.Once          // close closed channel once
	closed chan interface{}   // closed on Close
	decMu  sync.Mutex         // guards d
	d      *json.Decoder      // decodes incoming requests
	encMu  sync.Mutex         // guards e
	e      *json.Encoder      // encodes responses
	rw     io.ReadWriteCloser // connection
	req    *http.Request
}

// NewJSONCodec creates a new RPC server codec with support for JSON-RPC 2.0
func NewJSONCodec(rwc io.ReadWriteCloser, req *http.Request) *jsonCodecImpl {
	d := json.NewDecoder(rwc)
	d.UseNumber()
	return &jsonCodecImpl{
		closed: make(chan interface{}),
		d:      d,
		e:      json.NewEncoder(rwc),
		rw:     rwc,
		req:    req,
	}
}

//checkHttpHeader()

//checkAdminHttpHeader()

// ReadRawRequest will read new requests without parsing the arguments. It will
// return a collection of requests, an indication if these requests are in batch
// form or an error when the incoming message could not be read/parsed.
func (c *jsonCodecImpl) ReadRawRequest(options CodecOption) ([]*scommon.RPCRequest, bool, code.RPCError) {
	c.decMu.Lock()
	defer c.decMu.Unlock()

	var incomingMsg json.RawMessage
	if err := c.d.Decode(&incomingMsg); err != nil {
		return nil, false, &codeErr.InvalidRequestError{Message: err.Error()}
	}
	if b, err := incomingMsg.MarshalJSON(); err != nil {
		fmt.Printf("Got a request: %s", string(b))
	}
	if isBatch(incomingMsg) {
		return parseBatchRequest(incomingMsg)
	}

	return parseRequest(incomingMsg, options)
}

// GatAuthInfo read authentication info (token and method) from http header.
func (c *jsonCodecImpl) GetAuthInfo() (string, string) {
	token := c.req.Header.Get("Authorization")
	method := c.req.Header.Get("Method")
	return token, method
}

// isBatch returns true when the first non-whitespace characters is '['
func isBatch(msg json.RawMessage) bool {
	for _, c := range msg {
		// skip insignificant whitespace (http://www.ietf.org/rfc/rfc4627.txt)
		if c == 0x20 || c == 0x09 || c == 0x0a || c == 0x0d {
			continue
		}
		return c == '['
	}
	return false
}

// checkReqID returns an error when the given reqId isn't valid for RPC method calls.
// valid id's are strings, numbers or null
func checkReqID(reqID json.RawMessage) error {
	if len(reqID) == 0 {
		return errors.New("missing request id")
	}
	if _, err := strconv.ParseFloat(string(reqID), 64); err == nil {
		return nil
	}
	var str string
	if err := json.Unmarshal(reqID, &str); err == nil {
		return nil
	}
	return errors.New("invalid request id")
}

// parseRequest will parse a single request from the given RawMessage. It will return
// the parsed request, an indication if the request was a batch or an error when
// the request could not be parsed.
func parseRequest(incomingMsg json.RawMessage, options CodecOption) ([]*scommon.RPCRequest, bool, code.RPCError) {
	var in scommon.JSONRequest
	if err := json.Unmarshal(incomingMsg, &in); err != nil {
		return nil, false, &codeErr.InvalidMessageError{Message: err.Error()}
	}
	if err := checkReqID(in.ID); err != nil {
		return nil, false, &codeErr.InvalidMessageError{Message: err.Error()}
	}

	// regular RPC call
	elems := strings.Split(in.Method, codeErr.ServiceMethodSeparator)
	if len(elems) != 2 {
		return nil, false, &codeErr.MethodNotFoundError{Service: in.Method, Method: ""}
	}

	if len(in.Payload) == 0 {
		return []*scommon.RPCRequest{{Service: elems[0], Method: elems[1], ID: &in.ID}}, false, nil
	}

	return []*scommon.RPCRequest{{Service: elems[0], Method: elems[1], ID: &in.ID, Params: in.Payload}}, false, nil
}

// parseBatchRequest will parse a batch request into a collection of requests from the given RawMessage, an indication
// if the request was a batch or an error when the request could not be read.
func parseBatchRequest(incomingMsg json.RawMessage) ([]*scommon.RPCRequest, bool, code.RPCError) {
	var in []scommon.JSONRequest
	if err := json.Unmarshal(incomingMsg, &in); err != nil {
		return nil, false, &codeErr.InvalidMessageError{Message: err.Error()}
	}

	requests := make([]*scommon.RPCRequest, len(in))
	for i, r := range in {
		if err := checkReqID(r.ID); err != nil {
			return nil, false, &codeErr.InvalidMessageError{Message: err.Error()}
		}

		id := &in[i].ID

		elems := strings.Split(r.Method, codeErr.ServiceMethodSeparator)
		if len(elems) != 2 {
			return nil, true, &codeErr.MethodNotFoundError{Service: r.Method, Method: ""}
		}

		if len(r.Payload) == 0 {
			requests[i] = &scommon.RPCRequest{Service: elems[0], Method: elems[1], ID: id, Params: nil}
		} else {
			requests[i] = &scommon.RPCRequest{Service: elems[0], Method: elems[1], ID: id, Params: r.Payload}
		}
	}

	return requests, true, nil
}

// CreateResponse will create a JSON-RPC success response with the given id and reply as result.
func (c *jsonCodecImpl) CreateResponse(id interface{}, reply interface{}) interface{} {
	if isHexNum(reflect.TypeOf(reply)) {
		return &scommon.JSONResponse{Version: scommon.JSONRPCVersion, ID: id, Code: 0, Message: "SUCCESS", Result: fmt.Sprintf(`%#x`, reply)}
	}
	return &scommon.JSONResponse{Version: scommon.JSONRPCVersion, ID: id, Code: 0, Message: "SUCCESS", Result: reply}
}

// CreateErrorResponse will create a JSON-RPC error response with the given id and error.
func (c *jsonCodecImpl) CreateErrorResponse(id interface{}, err code.RPCError) interface{} {
	return &scommon.JSONResponse{Version: scommon.JSONRPCVersion, ID: id, Code: err.Code(), Message: err.Error()}
}

// CreateErrorResponseWithInfo will create a JSON-RPC error response with the given id and error.
// info is optional and contains additional information about the error. When an empty string is passed it is ignored.
func (c *jsonCodecImpl) CreateErrorResponseWithInfo(id interface{}, err code.RPCError, info interface{}) interface{} {
	return &scommon.JSONResponse{Version: scommon.JSONRPCVersion, ID: id, Code: err.Code(), Message: err.Error(), Info: info}
}

// CreateNotification will create a JSON-RPC notification with the given subscription id and event as params.
//func (c *jsonCodecImpl) CreateNotification(subid common.ID, service, method, namespace string, event interface{}) interface{}

// Write will write response to client.
func (c *jsonCodecImpl) Write(res interface{}) error {
	c.encMu.Lock()
	defer c.encMu.Unlock()

	return c.e.Encode(res)
}

// func (c *jsonCodecImpl) WriteNotify(res interface{}) error
// Close will close the underlying connection.
func (c *jsonCodecImpl) Close() {
	c.closer.Do(func() {
		close(c.closed)
		c.rw.Close()
	})
}

// Closed returns a channel which will be closed when Close is called.
func (c *jsonCodecImpl) Closed() <-chan interface{} {
	return c.closed
}
