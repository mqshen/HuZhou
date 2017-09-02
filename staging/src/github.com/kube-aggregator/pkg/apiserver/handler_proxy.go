package apiserver

import (
	"sync/atomic"
	genericapirequest "github.com/HuZhou/apiserver/pkg/endpoints/request"
	"net/http"
)

// proxyHandler provides a http.Handler which will proxy traffic to locations
// specified by items implementing Redirector.
type proxyHandler struct {
	contextMapper genericapirequest.RequestContextMapper

	// localDelegate is used to satisfy local APIServices
	localDelegate http.Handler

	// proxyClientCert/Key are the client cert used to identify this proxy. Backing APIServices use
	// this to confirm the proxy's identity
	proxyClientCert []byte
	proxyClientKey  []byte
	proxyTransport  *http.Transport

	// Endpoints based routing to map from cluster IP to routable IP
	serviceResolver ServiceResolver

	handlingInfo atomic.Value
}
