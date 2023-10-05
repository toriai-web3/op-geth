package code

import (
	"fmt"
)

// Common error message format
const (
	FmtNotFoundMethod      = "the method %s_%s does not exist/is not available"
	FmtNotFoundBlockNumber = "not found block number %#x"
	FmtNotFoundBlockHash   = "not found block hash %s"
	FmtNotFoundTxHash      = "not found transaction %s"
	FmtNotFoundNamespace   = "not found namespace %s"
	FmtNotFoundAccount     = "not found account %s"
	FmtNotFoundNode        = "not found node %v"
	FmtNotFoundReceipt     = "not found receipt %v"
	FmtNotFoundDiscardTx   = "not found discard transaction %v"
)

var _codes = make(map[int]string)

// New checks whether the code is unique or not.
func New(c int, message string) int {
	if _, ok := _codes[c]; ok {
		panic(fmt.Sprintf("error code: %d already exist", c))
	}
	_codes[c] = message
	return c
}

// RPCError implements RPC error, is add support for error codec over regular go errors
type RPCError interface {
	// RPC error code
	Code() int
	// Error message
	Error() string
}

type customError struct {
	code    int
	message string
}

func (ce *customError) Error() string {
	return ce.message
}

func (ce *customError) Code() int {
	return ce.code
}

// NewError creates and returns a new instance of customError.
func NewError(code int, format string, v ...interface{}) RPCError {
	desc, _ := _codes[code]
	err := &customError{code, desc}
	if format != "" {
		if err.message != "" {
			err.message += ": " + fmt.Sprintf(format, v...)
		} else {
			err.message = fmt.Sprintf(format, v...)
		}
	}
	return err
}
