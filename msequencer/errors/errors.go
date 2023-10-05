package common

import "fmt"

const (
	ServiceMethodSeparator = "_"
)

// JSON-RPC specified error
const (
	specified_InvalidMessageError int = -32700
	specified_CallbackError       int = -32000
	specified_ShutdownError       int = -32000
)
const (
	specified_InvalidRequestError int = -32600 - iota
	specified_MethodNotFoundError
	specified_InvalidParamsError
)

// JSON-RPC custom error code from 32001 to 32099
const (
	custom_InvalidTokenError int = -32097
	custom_UnauthorizedError int = -32098
	custom_CertError         int = -32099
)

const (
	custom_DBNotFoundError int = -32001 - iota
	custom_OutofBalanceError
	custom_SignatureInvalidError
	custom_ContractDeployError
	custom_ContractInvokeError
	custom_SystemTooBusyError
	custom_RepeatedTxError
	custom_ContractPermissionError
	custom_AccountNotExistError
	custom_NamespaceNotFoundError
	custom_NoBlockGeneratedError
	custom_SubNotExistError
	custom_SnapshotError
	custom_APINotFoundError
)

// RPCError implements RPC error, is add support for error codec over regular go errors
type RPCError interface {
	// RPC error code
	Code() int
	// Error message
	Error() string
}

// CORE ERRORS
// received message isn't a valid request
type InvalidRequestError struct {
	Message string
}

func (e *InvalidRequestError) Code() int     { return specified_InvalidRequestError }
func (e *InvalidRequestError) Error() string { return e.Message }

// request is for an unknown service
type MethodNotFoundError struct {
	Service string
	Method  string
}

func (e *MethodNotFoundError) Code() int { return specified_MethodNotFoundError }
func (e *MethodNotFoundError) Error() string {
	return fmt.Sprintf("The method %s%s%s does not exist/is not available",
		e.Service, ServiceMethodSeparator, e.Method)
}

// unable to decode supplied params, or an invalid number of parameters
type InvalidParamsError struct {
	Message string
}

func (e *InvalidParamsError) Code() int     { return specified_InvalidParamsError }
func (e *InvalidParamsError) Error() string { return e.Message }

// received message is invalid
type InvalidMessageError struct {
	Message string
}

func (e *InvalidMessageError) Code() int     { return specified_InvalidMessageError }
func (e *InvalidMessageError) Error() string { return e.Message }

// logic error, callback returned an error
type CallbackError struct {
	Message string
}

func (e *CallbackError) Code() int     { return specified_CallbackError }
func (e *CallbackError) Error() string { return e.Message }

// issued when a request is received after the server is issued to stop.
type ShutdownError struct{}

func (e *ShutdownError) Code() int     { return specified_ShutdownError }
func (e *ShutdownError) Error() string { return "Server is shutting down" }

// JSONRPC custom ERRORS
type DBNotFoundError struct {
	Type string
	ID   string
}

func (e *DBNotFoundError) Code() int     { return custom_DBNotFoundError }
func (e *DBNotFoundError) Error() string { return fmt.Sprintf("Not found %v %v", e.Type, e.ID) }

type OutOfBalanceError struct {
	Message string
}

func (e *OutOfBalanceError) Code() int     { return custom_OutofBalanceError }
func (e *OutOfBalanceError) Error() string { return e.Message }

type SignatureInvalidError struct {
	Message string
}

func (e *SignatureInvalidError) Code() int     { return custom_SignatureInvalidError }
func (e *SignatureInvalidError) Error() string { return e.Message }

type ContractDeployError struct {
	Message string
}

func (e *ContractDeployError) Code() int     { return custom_ContractDeployError }
func (e *ContractDeployError) Error() string { return e.Message }

type ContractInvokeError struct {
	Message string
}

func (e *ContractInvokeError) Code() int     { return custom_ContractInvokeError }
func (e *ContractInvokeError) Error() string { return e.Message }

type SystemTooBusyError struct{}

func (e *SystemTooBusyError) Code() int     { return custom_SystemTooBusyError }
func (e *SystemTooBusyError) Error() string { return "System is too busy to response." }

type RepeatedTxError struct {
	TxHash string
}

func (e *RepeatedTxError) Code() int     { return custom_RepeatedTxError }
func (e *RepeatedTxError) Error() string { return "Repeated transaction " + e.TxHash }

type ContractPermissionError struct {
	Message string
}

func (e *ContractPermissionError) Code() int { return custom_ContractPermissionError }
func (e *ContractPermissionError) Error() string {
	return fmt.Sprintf("The contract invocation permission not enough '%s'", e.Message)
}

type AccountNotExistError struct {
	Address string
}

func (e *AccountNotExistError) Code() int { return custom_AccountNotExistError }
func (e *AccountNotExistError) Error() string {
	return fmt.Sprintf("The account dose not exist '%s'", e.Address)
}

type NamespaceNotFound struct {
	Name string
}

func (e *NamespaceNotFound) Code() int { return custom_NamespaceNotFoundError }
func (e *NamespaceNotFound) Error() string {
	return fmt.Sprintf("The namespace '%s' does not exist", e.Name)
}

type NoBlockGeneratedError struct{}

func (e *NoBlockGeneratedError) Code() int     { return custom_NoBlockGeneratedError }
func (e *NoBlockGeneratedError) Error() string { return "There is no block generated!" }

type InvalidTokenError struct {
	Message string
}

func (e *InvalidTokenError) Code() int     { return custom_InvalidTokenError }
func (e *InvalidTokenError) Error() string { return fmt.Sprintf(e.Message) }

type SubNotExistError struct {
	Message string
}

func (e *SubNotExistError) Code() int     { return custom_SubNotExistError }
func (e *SubNotExistError) Error() string { return e.Message }

type SnapshotErr struct {
	Message string
}

func (e *SnapshotErr) Code() int     { return custom_SnapshotError }
func (e *SnapshotErr) Error() string { return e.Message }

type UnauthorizedError struct{}

func (e *UnauthorizedError) Code() int     { return custom_UnauthorizedError }
func (e *UnauthorizedError) Error() string { return "Unauthorized, Please check your cert" }

type CertError struct {
	Message string
}

func (e *CertError) Code() int     { return custom_CertError }
func (e *CertError) Error() string { return e.Message }

type APINotFoundError struct {
	Port    int
	Ip      string
	Message string
	Service string
	Method  string
}

func (e *APINotFoundError) Code() int { return custom_APINotFoundError }
func (e *APINotFoundError) Error() string {
	return fmt.Sprintf("%s_%s is served at %s:%d", e.Service, e.Method, e.Ip, e.Port)
}
