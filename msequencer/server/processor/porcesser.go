package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/msequencer/api"
	code "github.com/ethereum/go-ethereum/msequencer/errors/errorcode"
	"reflect"
	"strconv"
	"sync"

	codeErr "github.com/ethereum/go-ethereum/msequencer/errors"
	scommon "github.com/ethereum/go-ethereum/msequencer/server/common"
)

type Processor struct {
	innerProcessor RequestProcessor
}

func NewProcessor() (*Processor, error) {
	processor := &Processor{}
	apis := processor.GetApis()
	processor.innerProcessor = NewJsonRpcProcessorImpl(apis)
	err := processor.innerProcessor.Start()
	return processor, err
}

func (p *Processor) ProcessRequest(req *scommon.RPCRequest) interface{} {
	return p.innerProcessor.ProcessRequest(req)
}

func (p *Processor) GetApis() map[string]*api.API {
	return map[string]*api.API{
		"test": {
			Svcname: "test",
			Version: "1.0",
			Service: api.NewTest(),
			Public:  true,
		},
		"tx": {
			Svcname: "tx",
			Version: "1.0",
			Service: api.NewTransaction(),
			Public:  true,
		},
	}
}

type RequestProcessor interface {
	// Start registers all the JSON-RPC API service.
	Start() error

	// Stop stops process request.(Need to be implemented later.)
	Stop() error

	// ProcessRequest checks request parameters and then executes the given request.
	ProcessRequest(request *scommon.RPCRequest) *scommon.RPCResponse
}

type serviceRegistry map[string]*service // collection of services

type JsonRpcProcessorImpl struct {
	// apis this namespace provides.
	apis map[string]*api.API

	// register the apis of this namespace to this processor.
	services serviceRegistry

	// registerLocker makes the register activities thread-safe
	registerLocker sync.Mutex
}

// NewJsonRpcProcessorImpl creates a new JsonRpcProcessorImpl instance for given namespace and apis.
func NewJsonRpcProcessorImpl(apis map[string]*api.API) *JsonRpcProcessorImpl {
	jpri := &JsonRpcProcessorImpl{
		apis:     apis,
		services: make(serviceRegistry),
	}
	return jpri
}

func (jrpi *JsonRpcProcessorImpl) Start() error {
	err := jrpi.registerAllAPIService()
	if err != nil {
		return err
	}
	return nil
}

func (jrpi *JsonRpcProcessorImpl) Stop() error {
	return nil
}

func (jrpi *JsonRpcProcessorImpl) ProcessRequest(request *scommon.RPCRequest) *scommon.RPCResponse {
	sr := jrpi.checkRequestParams(request)
	return jrpi.exec(request.Ctx, sr)
}

// exec executes the given request and returns the response to upper layer.
func (jrpi *JsonRpcProcessorImpl) exec(ctx context.Context, req *serverRequest) *scommon.RPCResponse {

	response, callback := jrpi.handle(ctx, req)

	// when request was a subscribe request this allows these subscriptions to be active.
	if callback != nil {
		callback()
	}
	return response
}

// handle executes a request and returns the response from the method callback.
func (jrpi *JsonRpcProcessorImpl) handle(ctx context.Context, req *serverRequest) (*scommon.RPCResponse, func()) {
	if req.err != nil {
		return jrpi.CreateErrorResponse(&req.id, req.err), nil
	}

	// regular RPC call, prepare arguments
	if len(req.args) != len(req.callb.argTypes) {
		errMsg := fmt.Sprintf("%s%s%s expects %d parameters, got %d",
			req.svcname, "_", req.callb.method.Name, len(req.callb.argTypes), len(req.args))
		err := &codeErr.InvalidParamsError{Message: errMsg}
		return jrpi.CreateErrorResponse(&req.id, err), nil
	}

	arguments := []reflect.Value{req.callb.rcvr}
	if req.callb.hasCtx {
		arguments = append(arguments, reflect.ValueOf(ctx))
	}
	if len(req.args) > 0 {
		arguments = append(arguments, req.args...)
	}

	// execute RPC method and return result
	reply := req.callb.method.Func.Call(arguments)
	if len(reply) == 0 {
		return jrpi.CreateResponse(req.id, nil), nil
	}

	// test if method returned an error
	if req.callb.errPos >= 0 {
		if !reply[req.callb.errPos].IsNil() {
			e := reply[req.callb.errPos].Interface().(code.RPCError)
			if !isEmpty(reply[0]) {
				return jrpi.CreateErrorResponseWithInfo(&req.id, e, reply[0].Interface()), nil
			}
			return jrpi.CreateErrorResponse(&req.id, e), nil
		}
	}
	return jrpi.CreateResponse(req.id, reply[0].Interface()), nil
}

func isEmpty(v reflect.Value) bool {
	k := v.Kind()
	switch k {
	case reflect.String:
		return v.String() == ""
	case reflect.Ptr, reflect.Chan, reflect.Func, reflect.Map, reflect.Interface, reflect.Slice:
		return v.IsNil()
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10) == "0"
	case reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10) == "0"
	case reflect.Bool:
		return v.Bool() == false
	default:
		if addr, ok := v.Interface().(common.Address); ok {
			return len(addr.String()) == 0
		} else if hash, ok := v.Interface().(common.Hash); ok {
			return len(hash.String()) == 0
		}
		return true
	}
}

// CreateResponse will create regular RPCResponse.
func (jrpi *JsonRpcProcessorImpl) CreateResponse(id interface{}, reply interface{}) *scommon.RPCResponse {
	if _, ok := reply.(ID); ok {
		return &scommon.RPCResponse{
			ID:       id,
			Reply:    reply,
			Error:    nil,
			IsPubSub: true,
		}
	}

	return &scommon.RPCResponse{
		ID:       id,
		Reply:    reply,
		Error:    nil,
		IsPubSub: false,
	}
}

// CreateErrorResponse will create an error RPCResponse.
func (jrpi *JsonRpcProcessorImpl) CreateErrorResponse(id interface{}, err code.RPCError) *scommon.RPCResponse {
	return &scommon.RPCResponse{
		ID:    id,
		Reply: nil,
		Error: err,
	}
}

// CreateErrorResponseWithInfo will create an error RPCResponse with given info.
func (jrpi *JsonRpcProcessorImpl) CreateErrorResponseWithInfo(id interface{}, err code.RPCError, info interface{}) *scommon.RPCResponse {
	return &scommon.RPCResponse{
		ID:    id,
		Reply: info,
		Error: err,
	}
}

// checkRequestParams requests the next (batch) request from the codec. It will return the collection
// of requests, an indication if the request was a batch, the invalid request identifier and an
// error when the request could not be read/parsed.
func (jrpi *JsonRpcProcessorImpl) checkRequestParams(req *scommon.RPCRequest) *serverRequest {
	var sr *serverRequest
	var ok bool
	var svc *service

	// If the given rpc method isn't available, return error.
	if svc, ok = jrpi.services[req.Service]; !ok {
		sr = &serverRequest{id: req.ID, err: &codeErr.MethodNotFoundError{Service: req.Service, Method: req.Method}}
		return sr
	}

	// For sub_subscribe, req.method contains the subscription method name.
	if req.IsPubSub {
		sr = &serverRequest{id: req.ID, err: &codeErr.MethodNotFoundError{Service: req.Service, Method: req.Method}}
		return sr
	}

	// For callbacks, req.method contains the callback method name, lookup RPC method.
	if callb, ok := svc.callbacks[req.Method]; ok {
		sr = &serverRequest{id: req.ID, svcname: svc.name, callb: callb}
		if req.Params != nil && len(callb.argTypes) > 0 {
			if args, err := jrpi.parseRequestArguments(callb.argTypes, req.Params); err == nil {
				sr.args = args
			} else {
				sr.err = &codeErr.InvalidParamsError{Message: err.Error()}
			}
		}
		return sr
	}

	return &serverRequest{
		id: req.ID,
		err: &codeErr.MethodNotFoundError{
			Service: req.Service,
			Method:  req.Method},
	}

}

// parseRequestArguments tries to parse the given params (json.RawMessage) with the given
// types. It returns the parsed values or an error when the parsing failed.
func (jrpi *JsonRpcProcessorImpl) parseRequestArguments(argTypes []reflect.Type, params interface{}) ([]reflect.Value, error) {
	if args, ok := params.(json.RawMessage); !ok {
		return nil, &codeErr.InvalidParamsError{
			Message: "Invalid params supplied",
		}
	} else {
		return jrpi.parsePositionalArguments(args, argTypes)
	}
}

// parsePositionalArguments tries to parse the given args to an array of values with the
// given types. It returns the parsed values or an error when the args could not be parsed.
// Missing optional arguments are returned as reflect.Zero values.
func (jrpi *JsonRpcProcessorImpl) parsePositionalArguments(args json.RawMessage, callbackArgs []reflect.Type) ([]reflect.Value, error) {
	params := make([]interface{}, 0, len(callbackArgs))

	for _, t := range callbackArgs {
		params = append(params, reflect.New(t).Interface())
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, &codeErr.InvalidParamsError{Message: err.Error()}
	}

	if len(params) > len(callbackArgs) {
		errMsg := fmt.Sprintf("too many params, want %d got %d", len(callbackArgs), len(params))
		return nil, &codeErr.InvalidParamsError{Message: errMsg}
	}

	// assume missing params are null values
	for i := len(params); i < len(callbackArgs); i++ {
		params = append(params, nil)
	}

	argValues := make([]reflect.Value, len(params))
	for i, p := range params {
		// verify that JSON null values are only supplied for optional arguments (ptr types)
		if p == nil && callbackArgs[i].Kind() != reflect.Ptr {
			errMsg := fmt.Sprintf("invalid or missing value for params[%d]", i)
			return nil, &codeErr.InvalidParamsError{Message: errMsg}
		}
		if p == nil {
			argValues[i] = reflect.Zero(callbackArgs[i])
		} else {
			// deref pointers values creates previously with reflect.New
			argValues[i] = reflect.ValueOf(p).Elem()
		}
	}
	return argValues, nil
}

// registerAllAPIService will register all the JSON-RPC API service. If there are
// no services offered, an error is returned.
func (jrpi *JsonRpcProcessorImpl) registerAllAPIService() error {
	if jrpi.apis == nil || len(jrpi.apis) == 0 {
		return scommon.ErrNoApis
	}
	jrpi.registerLocker.Lock()
	defer jrpi.registerLocker.Unlock()
	for _, api := range jrpi.apis {
		if err := jrpi.registerAPIService(api.Svcname, api.Service); err != nil {
			return err
		}
	}

	return nil
}

// registerAPIService will create a service for the given rcvr type under the given svcname.
// When no methods on the given rcvr match the criteria to be either a RPC method or a
// subscription, then an error is returned. Otherwise a new service is created and added to
// the service collection this server instance serves.
func (jrpi *JsonRpcProcessorImpl) registerAPIService(svcname string, rcvr interface{}) error {
	svc := new(service)
	svc.typ = reflect.TypeOf(rcvr)
	rcvrVal := reflect.ValueOf(rcvr)

	if svcname == "" {
		return scommon.ErrNoServiceName
	}

	if !isExported(reflect.Indirect(rcvrVal).Type().Name()) {
		return scommon.ErrNotExported
	}

	svc.name = svcname
	callbacks, subscriptions := suitableCallbacks(rcvrVal, svc.typ)
	if len(callbacks) == 0 && len(subscriptions) == 0 {
		return scommon.ErrNoSuitable
	}

	// If there already existed a previous service registered under the given service name,
	// merge the methods/subscriptions, else use the service newed before.
	if regsvc, present := jrpi.services[svcname]; present {
		for _, m := range callbacks {
			regsvc.callbacks[formatName(m.method.Name)] = m
		}
		for _, s := range subscriptions {
			regsvc.subscriptions[formatName(s.method.Name)] = s
		}
		return nil
	}

	svc.callbacks, svc.subscriptions = callbacks, subscriptions
	jrpi.services[svc.name] = svc
	return nil
}
