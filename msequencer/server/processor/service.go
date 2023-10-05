package processor

import (
	code "github.com/ethereum/go-ethereum/msequencer/errors/errorcode"
	"reflect"
)

// service represents a registered object
type service struct {
	// name for service
	name string

	// receiver of methods for the service
	rcvr reflect.Value

	// receiver type, such as *Block, *Transaction in api
	typ reflect.Type

	// registered normal handlers under this receiver
	callbacks callbacks

	// available subscriptions/notifications under this receiver
	subscriptions subscriptions
}

type callbacks map[string]*callback     // collection of RPC callbacks
type subscriptions map[string]*callback // collection of subscription callbacks

// callback is a method callback which was registered in the server
type callback struct {
	// receiver of the method
	rcvr reflect.Value

	// a specific callback method
	method reflect.Method

	// input argument types exclude receiver and optional context
	argTypes []reflect.Type

	// method's first argument is a context (not included in argTypes) or not
	hasCtx bool

	// err return idx, of -1 when method cannot return error, or of the last index
	// if has err return
	errPos int

	// indication if the callback is a subscription
	isSubscribe bool
}

// serverRequest is an incoming request
type serverRequest struct {
	// id equals the request.ID
	id interface{}
	// svcname is the service.name, such as block, contract...
	svcname string

	rcvr reflect.Value

	// callb is the certain callback of a specified method.
	callb *callback

	// args is the parsed arguments of this method.
	args []reflect.Value

	// If a request is a subscribe request, isUnsubscribe is true means it is a unsubscribe request.
	isUnsubscribe bool

	// error value.
	err code.RPCError
}
