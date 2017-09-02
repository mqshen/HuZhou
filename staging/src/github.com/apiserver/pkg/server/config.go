package server

import (
	genericregistry "github.com/HuZhou/apiserver/pkg/registry/generic"
	genericfilters "github.com/HuZhou/apiserver/pkg/server/filters"
	genericapifilters "github.com/HuZhou/apiserver/pkg/endpoints/filters"
	apirequest "github.com/HuZhou/apiserver/pkg/endpoints/request"
	auditpolicy "github.com/HuZhou/apiserver/pkg/audit/policy"
	restclient "k8s.io/client-go/rest"
	"k8s.io/apimachinery/pkg/version"
	"github.com/HuZhou/apiserver/pkg/audit"
	"io"
	"time"
	"k8s.io/apimachinery/pkg/util/sets"
	"strings"
	"crypto/x509"
	"crypto/tls"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	"github.com/HuZhou/apiserver/pkg/authorization/authorizer"
	"github.com/HuZhou/apiserver/pkg/authentication/authenticator"
	"github.com/HuZhou/apiserver/pkg/server/routes"
)

const (
	// DefaultLegacyAPIPrefix is where the the legacy APIs will be located.
	DefaultLegacyAPIPrefix = "/api"

	// APIGroupPrefix is where non-legacy API group will be located.
	APIGroupPrefix = "/apis"
)

type Config struct {
	// LoopbackClientConfig is a config for a privileged loopback connection to the API server
	// This is required for proper functioning of the PostStartHooks on a GenericAPIServer
	LoopbackClientConfig *restclient.Config
	Authenticator authenticator.Request

	Authorizer authorizer.Authorizer
	// Serializer is required and provides the interface for serializing and converting objects to and from the wire
	// The default (api.Codecs) usually works fine.
	Serializer runtime.NegotiatedSerializer
	// TODO(roberthbailey): Remove once the server no longer supports http basic auth.
	SupportsBasicAuth bool

	EnableDiscovery bool

	EnableIndex     bool
	// RESTOptionsGetter is used to construct RESTStorage types via the generic registry.
	RESTOptionsGetter genericregistry.RESTOptionsGetter

	// Version will enable the /version endpoint if non-nil
	Version *version.Info

	RequestContextMapper apirequest.RequestContextMapper
	// AuditBackend is where audit events are sent to.
	AuditBackend audit.Backend// AuditPolicyChecker makes the decision of whether and how to audit log a request.
	AuditPolicyChecker auditpolicy.Checker// Predicate which is true for paths of long-running http requests
	LongRunningFunc apirequest.LongRunningRequestCheck
	// LegacyAuditWriter is the destination for audit logs. If nil, they will not be written.
	LegacyAuditWriter io.Writer

	CorsAllowedOriginList []string

	// If specified, all requests except those which match the LongRunningFunc predicate will timeout
	// after this duration.
	RequestTimeout time.Duration

	// MaxRequestsInFlight is the maximum number of parallel non-long-running requests. Every further
	// request has to wait. Applies only to non-mutating requests.
	MaxRequestsInFlight int
	// MaxMutatingRequestsInFlight is the maximum number of parallel mutating requests. Every further
	// request has to wait.
	MaxMutatingRequestsInFlight int

	// LegacyAPIGroupPrefixes is used to set up URL parsing for authorization and for validating requests
	// to InstallLegacyAPIGroup. New API servers don't generally have legacy groups at all.
	LegacyAPIGroupPrefixes sets.String

	// BuildHandlerChainFunc allows you to build custom handler chains by decorating the apiHandler.
	BuildHandlerChainFunc func(apiHandler http.Handler, c *Config) (secure http.Handler)
}


type completedConfig struct {
	*Config
}

// Complete fills in any fields not set that are required to have valid data and can be derived
// from other fields. If you're going to `ApplyOptions`, do that first. It's mutating the receiver.
func (c *Config) Complete() completedConfig {

	return completedConfig{c}
}

// SkipComplete provides a way to construct a server instance without config completion.
func (c *Config) SkipComplete() completedConfig {
	return completedConfig{c}
}

// New creates a new server which logically combines the handling chain with the passed server.
// name is used to differentiate for logging. The handler chain in particular can be difficult as it starts delgating.
func (c completedConfig) New(name string, delegationTarget DelegationTarget) (*GenericAPIServer, error) {
	handlerChainBuilder := func(handler http.Handler) http.Handler {
		return c.BuildHandlerChainFunc(handler, c.Config)
	}
	apiServerHandler := NewAPIServerHandler(name, c.RequestContextMapper, c.Serializer, handlerChainBuilder, delegationTarget.UnprotectedHandler())

	s := &GenericAPIServer{
		postStartHooks:         map[string]postStartHookEntry{},
		Handler: 				apiServerHandler,
		listedPathProvider: 	apiServerHandler,
	}
	installAPI(s, c.Config)
	return s, nil
}

func installAPI(s *GenericAPIServer, c *Config) {
	//if c.EnableIndex {
		routes.Index{}.Install(s.listedPathProvider, s.Handler.NonGoRestfulMux)
	//}
	//if c.SwaggerConfig != nil && c.EnableSwaggerUI {
	//	routes.SwaggerUI{}.Install(s.Handler.NonGoRestfulMux)
	//}
	//if c.EnableProfiling {
	//	routes.Profiling{}.Install(s.Handler.NonGoRestfulMux)
	//	if c.EnableContentionProfiling {
	//		goruntime.SetBlockProfileRate(1)
	//	}
	//}
	//if c.EnableMetrics {
	//	if c.EnableProfiling {
	//		routes.MetricsWithReset{}.Install(s.Handler.NonGoRestfulMux)
	//	} else {
	//		routes.DefaultMetrics{}.Install(s.Handler.NonGoRestfulMux)
	//	}
	//}
	routes.Version{Version: c.Version}.Install(s.Handler.GoRestfulContainer)

	//if c.EnableDiscovery {
	//	s.Handler.GoRestfulContainer.Add(s.DiscoveryGroupManager.WebService())
	//}
}

func NewRequestInfoResolver(c *Config) *apirequest.RequestInfoFactory {
	apiPrefixes := sets.NewString(strings.Trim(APIGroupPrefix, "/")) // all possible API prefixes
	legacyAPIPrefixes := sets.String{}                               // APIPrefixes that won't have groups (legacy)
	for legacyAPIPrefix := range c.LegacyAPIGroupPrefixes {
		apiPrefixes.Insert(strings.Trim(legacyAPIPrefix, "/"))
		legacyAPIPrefixes.Insert(strings.Trim(legacyAPIPrefix, "/"))
	}

	return &apirequest.RequestInfoFactory{
		APIPrefixes:          apiPrefixes,
		GrouplessAPIPrefixes: legacyAPIPrefixes,
	}
}


type SecureServingInfo struct {
	// BindAddress is the ip:port to serve on
	BindAddress string
	// BindNetwork is the type of network to bind to - defaults to "tcp", accepts "tcp",
	// "tcp4", and "tcp6".
	BindNetwork string

	// Cert is the main server cert which is used if SNI does not match. Cert must be non-nil and is
	// allowed to be in SNICerts.
	Cert *tls.Certificate

	// CACert is an optional certificate authority used for the loopback connection of the Admission controllers.
	// If this is nil, the certificate authority is extracted from Cert or a matching SNI certificate.
	CACert *tls.Certificate

	// SNICerts are the TLS certificates by name used for SNI.
	SNICerts map[string]*tls.Certificate

	// ClientCA is the certificate bundle for all the signers that you'll recognize for incoming client certificates
	ClientCA *x509.CertPool

	// MinTLSVersion optionally overrides the minimum TLS version supported.
	// Values are from tls package constants (https://golang.org/pkg/crypto/tls/#pkg-constants).
	MinTLSVersion uint16

	// CipherSuites optionally overrides the list of allowed cipher suites for the server.
	// Values are from tls package constants (https://golang.org/pkg/crypto/tls/#pkg-constants).
	CipherSuites []uint16
}

func DefaultBuildHandlerChain(apiHandler http.Handler, c *Config) http.Handler {
	handler := genericapifilters.WithAuthorization(apiHandler, c.RequestContextMapper, c.Authorizer, c.Serializer)
	handler = genericfilters.WithMaxInFlightLimit(handler, c.MaxRequestsInFlight, c.MaxMutatingRequestsInFlight, c.RequestContextMapper, c.LongRunningFunc)
	handler = genericapifilters.WithImpersonation(handler, c.RequestContextMapper, c.Authorizer, c.Serializer)
//	if utilfeature.DefaultFeatureGate.Enabled(features.AdvancedAuditing) {
	handler = genericapifilters.WithAudit(handler, c.RequestContextMapper, c.AuditBackend, c.AuditPolicyChecker, c.LongRunningFunc)
	//} else {
	//	handler = genericapifilters.WithLegacyAudit(handler, c.RequestContextMapper, c.LegacyAuditWriter)
	//}
	handler = genericapifilters.WithAuthentication(handler, c.RequestContextMapper, c.Authenticator, genericapifilters.Unauthorized(c.RequestContextMapper, c.Serializer, c.SupportsBasicAuth))
	handler = genericfilters.WithCORS(handler, c.CorsAllowedOriginList, nil, nil, nil, "true")
	handler = genericfilters.WithTimeoutForNonLongRunningRequests(handler, c.RequestContextMapper, c.LongRunningFunc, c.RequestTimeout)
	handler = genericapifilters.WithRequestInfo(handler, NewRequestInfoResolver(c), c.RequestContextMapper)
	handler = apirequest.WithRequestContext(handler, c.RequestContextMapper)
	handler = genericfilters.WithPanicRecovery(handler)
	return handler
}

// NewConfig returns a Config struct with the default values
func NewConfig(codecs serializer.CodecFactory) *Config {
	return &Config{
		Serializer:                   codecs,
		//ReadWritePort:                443,
		RequestContextMapper:         apirequest.NewRequestContextMapper(),
		BuildHandlerChainFunc:        DefaultBuildHandlerChain,
		LegacyAPIGroupPrefixes:       sets.NewString(DefaultLegacyAPIPrefix),
		//DisabledPostStartHooks:       sets.NewString(),
		//HealthzChecks:                []healthz.HealthzChecker{healthz.PingHealthz},
		//EnableIndex:                  true,
		EnableDiscovery:              true,
		//EnableProfiling:              true,
		MaxRequestsInFlight:          400,
		MaxMutatingRequestsInFlight:  200,
		RequestTimeout:               time.Duration(60) * time.Second,
		//MinRequestTimeout:            1800,
		//EnableAPIResponseCompression: utilfeature.DefaultFeatureGate.Enabled(features.APIResponseCompression),

		// Default to treating watch as a long-running operation
		// Generic API servers have no inherent long-running subresources
		LongRunningFunc: genericfilters.BasicLongRunningRequestCheck(sets.NewString("watch"), sets.NewString()),
	}
}