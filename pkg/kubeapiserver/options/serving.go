package options

import (
	"net"
	"github.com/HuZhou/apiserver/pkg/server"
	"strconv"
	"github.com/pborman/uuid"
	kubeserver "github.com/mqshen/HuZhou/pkg/kubeapiserver/server"
)

// InsecureServingOptions are for creating an unauthenticated, unauthorized, insecure port.
// No one should be using these anymore.
type InsecureServingOptions struct {
	BindAddress net.IP
	BindPort    int
}
// NewInsecureServingOptions is for creating an unauthenticated, unauthorized, insecure port.
// No one should be using these anymore.
func NewInsecureServingOptions() *InsecureServingOptions {
	return &InsecureServingOptions{
		BindAddress: net.ParseIP("127.0.0.1"),
		BindPort:    8080,
	}
}

func (s *InsecureServingOptions) ApplyTo(c *server.Config) (*kubeserver.InsecureServingInfo, error) {
	if s.BindPort <= 0 {
		return nil, nil
	}

	ret := &kubeserver.InsecureServingInfo{
		BindAddress: net.JoinHostPort(s.BindAddress.String(), strconv.Itoa(s.BindPort)),
	}

	var err error
	privilegedLoopbackToken := uuid.NewRandom().String()
	if c.LoopbackClientConfig, err = ret.NewLoopbackClientConfig(privilegedLoopbackToken); err != nil {
		return nil, err
	}

	return ret, nil
}