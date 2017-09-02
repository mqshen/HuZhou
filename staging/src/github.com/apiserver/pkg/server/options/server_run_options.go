package options

import (
	"net"
	"time"
	"github.com/HuZhou/apiserver/pkg/server"
)

// ServerRunOptions contains the options while running a generic api server.
type ServerRunOptions struct {
	AdvertiseAddress net.IP

	CorsAllowedOriginList       []string
	ExternalHost                string
	MaxRequestsInFlight         int
	MaxMutatingRequestsInFlight int
	RequestTimeout              time.Duration
	MinRequestTimeout           int
	TargetRAMMB                 int
	WatchCacheSizes             []string
}

// ApplyOptions applies the run options to the method receiver and returns self
func (s *ServerRunOptions) ApplyTo(c *server.Config) error {
	c.CorsAllowedOriginList = s.CorsAllowedOriginList
	//c.ExternalAddress = s.ExternalHost
	c.MaxRequestsInFlight = s.MaxRequestsInFlight
	c.MaxMutatingRequestsInFlight = s.MaxMutatingRequestsInFlight
	c.RequestTimeout = s.RequestTimeout
	//c.MinRequestTimeout = s.MinRequestTimeout
	//c.PublicAddress = s.AdvertiseAddress

	return nil
}
