package apiserver

import (
	"net/http"
	genericapiserver "github.com/HuZhou/apiserver/pkg/server"
	genericapirequest "github.com/HuZhou/apiserver/pkg/endpoints/request"
	listers "github.com/HuZhou/kube-aggregator/pkg/client/listers/apiregistration/internalversion"
	"k8s.io/client-go/informers"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/pkg/version"
)

type Config struct {
	GenericConfig *genericapiserver.Config

	// ProxyClientCert/Key are the client cert used to identify this proxy. Backing APIServices use
	// this to confirm the proxy's identity
	ProxyClientCert []byte
	ProxyClientKey  []byte

	// If present, the Dial method will be used for dialing out to delegate
	// apiservers.
	ProxyTransport *http.Transport

	// Mechanism by which the Aggregator will resolve services. Required.
	ServiceResolver ServiceResolver
}


// APIAggregator contains state for a Kubernetes cluster master/api server.
type APIAggregator struct {
	GenericAPIServer *genericapiserver.GenericAPIServer

	delegateHandler http.Handler

	contextMapper genericapirequest.RequestContextMapper

	// proxyClientCert/Key are the client cert used to identify this proxy. Backing APIServices use
	// this to confirm the proxy's identity
	proxyClientCert []byte
	proxyClientKey  []byte
	proxyTransport  *http.Transport

	// proxyHandlers are the proxy handlers that are currently registered, keyed by apiservice.name
	proxyHandlers map[string]*proxyHandler
	// handledGroups are the groups that already have routes
	handledGroups sets.String

	// lister is used to add group handling for /apis/<group> aggregator lookups based on
	// controller state
	lister listers.APIServiceLister

	// provided for easier embedding
	APIRegistrationInformers informers.SharedInformerFactory

	// Information needed to determine routing for the aggregator
	serviceResolver ServiceResolver

	openAPIAggregator *openAPIAggregator
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (c *Config) Complete() completedConfig {
	// the kube aggregator wires its own discovery mechanism
	// TODO eventually collapse this by extracting all of the discovery out
	c.GenericConfig.EnableDiscovery = false
	c.GenericConfig.Complete()

	version := version.Get()
	c.GenericConfig.Version = &version

	return completedConfig{c}
}

type completedConfig struct {
	*Config
}

// New returns a new instance of APIAggregator from the given config.
func (c completedConfig) NewWithDelegate(delegationTarget genericapiserver.DelegationTarget) (*APIAggregator, error) {
	genericServer, err := c.Config.GenericConfig.SkipComplete().New("kube-aggregator", delegationTarget) // completion is done in Complete, no need for a second time
	if err != nil {
		return nil, err
	}
	s := &APIAggregator{
		GenericAPIServer: genericServer,
		delegateHandler:  delegationTarget.UnprotectedHandler(),
		contextMapper:    c.GenericConfig.RequestContextMapper,
		proxyClientCert:  c.ProxyClientCert,
		proxyClientKey:   c.ProxyClientKey,
		proxyTransport:   c.ProxyTransport,
		proxyHandlers:    map[string]*proxyHandler{},
		handledGroups:    sets.String{},
		serviceResolver:          c.ServiceResolver,
	}
	return s, nil
}