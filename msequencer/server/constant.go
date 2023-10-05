package server

import "time"

const (
	maxHTTPRequestContentLength = 8 * 1024 * 1024
	readTimeout                 = 360 * time.Second
)

const (
	stopPendingRequestTimeout = 3 * time.Second // give pending requests stopPendingRequestTimeout the time to finish when the server is stopped
)

type CodecOption int

const (
	// OptionMethodInvocation is an indication that the codec supports RPC method calls
	OptionMethodInvocation CodecOption = 1 << iota

	// OptionSubscriptions is an indication that the codec suports RPC notifications
	OptionSubscriptions
)
