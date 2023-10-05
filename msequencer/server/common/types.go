package common

import (
	"context"
	"encoding/json"
	code "github.com/ethereum/go-ethereum/msequencer/errors/errorcode"
)

const (
	// JSONRPCVersion represents JSON-RPC version
	JSONRPCVersion = "2.0"
)

// RPCRequest represents a raw incoming RPC request
type RPCRequest struct {
	Service  string
	Method   string
	ID       interface{}
	IsPubSub bool
	Params   interface{}
	Ctx      context.Context
}

// RPCResponse represents a raw incoming RPC request
type RPCResponse struct {
	ID       interface{}
	Reply    interface{}
	Error    code.RPCError
	IsPubSub bool
	IsUnsub  bool
}

// JSONRequest describes a JSON-RPC request
type JSONRequest struct {
	Method  string          `json:"method"`
	Version string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Payload json.RawMessage `json:"params,omitempty"`
}

// JSONResponse describers a JSON-RPC response
type JSONResponse struct {
	Version string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
	Info    interface{} `json:"info,omitempty"`
}
