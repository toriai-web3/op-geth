package server

import (
	code "github.com/ethereum/go-ethereum/msequencer/errors/errorcode"
	scommon "github.com/ethereum/go-ethereum/msequencer/server/common"
)

// ServerCodec implements reading, parsing and writing RPC messages for the server side of
// a RPC session. Implementations must be go-routine safe since the codec can be called in
// multiple go-routines concurrently.
type ServerCodec interface {
	ReadRawRequest(options CodecOption) ([]*scommon.RPCRequest, bool, code.RPCError)
	CreateResponse(id interface{}, reply interface{}) interface{}
	CreateErrorResponse(id interface{}, err code.RPCError) interface{}
	CreateErrorResponseWithInfo(id interface{}, err code.RPCError, info interface{}) interface{}
	GetAuthInfo() (string, string)
	// Write msg to client.
	Write(interface{}) error
	// Close underlying data stream
	Close()
	// Closed when underlying connection is closed
	Closed() <-chan interface{}
}

// receiver implements handling RPC request in channel.
type receiver interface {
	handleChannelReq(codec ServerCodec, rq *scommon.RPCRequest) interface{}
}
