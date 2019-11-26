package netter

import (
	"net"
	"net/http"
	"time"
)

// DefaultTransport ..
var DefaultTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		// timeout on TCP dialing
		Timeout: 30 * time.Second,
		// TCP KeepAlive specifies the interval between keep-alive probes for an active network connection.
		KeepAlive: 30 * time.Second,
		// for both ip4 and ip6
		DualStack: true,
	}).DialContext,
	ResponseHeaderTimeout: 60 * time.Second,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	DisableKeepAlives:     true,
	MaxIdleConnsPerHost:   -1,
}
