package common

import "errors"

var (
	ErrNoApis        = errors.New("no api service will be registered")
	ErrNoServiceName = errors.New("no service name")
	ErrNotExported   = errors.New("not exported receiver")
	ErrNoSuitable    = errors.New("no suitable methods/subscriptions to expose")
	ErrInvalidReply  = errors.New("invalid subscription reply")
)
