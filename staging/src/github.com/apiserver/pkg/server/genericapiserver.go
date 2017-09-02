package server

import (
	systemd "github.com/coreos/go-systemd/daemon"
	"k8s.io/apimachinery/pkg/util/sets"
	apirequest "github.com/HuZhou/apiserver/pkg/endpoints/request"
	"sync"
	"github.com/HuZhou/apiserver/pkg/server/healthz"
	"net/http"
	"github.com/HuZhou/apiserver/pkg/server/routes"
	"github.com/emicklei/go-restful-swagger12"
	openapicommon "k8s.io/kube-openapi/pkg/common"
	"fmt"
	"github.com/golang/glog"
	"github.com/HuZhou/apiserver/pkg/audit"
	restclient "k8s.io/client-go/rest"
	"github.com/HuZhou/apiserver/pkg/endpoints/discovery"
)

var EmptyDelegate = emptyDelegate{
	requestContextMapper: apirequest.NewRequestContextMapper(),
}

type emptyDelegate struct {
	requestContextMapper apirequest.RequestContextMapper
}

func (s emptyDelegate) UnprotectedHandler() http.Handler {
	return nil
}
func (s emptyDelegate) PostStartHooks() map[string]postStartHookEntry {
	return map[string]postStartHookEntry{}
}
func (s emptyDelegate) HealthzChecks() []healthz.HealthzChecker {
	return []healthz.HealthzChecker{}
}
func (s emptyDelegate) ListedPaths() []string {
	return []string{}
}
func (s emptyDelegate) RequestContextMapper() apirequest.RequestContextMapper {
	return s.requestContextMapper
}

type postStartHookEntry struct {
	hook PostStartHookFunc

	// done will be closed when the postHook is finished
	done chan struct{}
}

type GenericAPIServer struct {

	// LoopbackClientConfig is a config for a privileged loopback connection to the API server
	LoopbackClientConfig *restclient.Config
	// Enable swagger and/or OpenAPI if these configs are non-nil.
	swaggerConfig *swagger.Config
	openAPIConfig *openapicommon.Config
	// "Outputs"
	// Handler holds the handlers being used by this API server
	Handler *APIServerHandler
	// listedPathProvider is a lister which provides the set of paths to show at /
	listedPathProvider routes.ListedPathProvider

	// DiscoveryGroupManager serves /apis
	DiscoveryGroupManager discovery.GroupManager

	// auditing. The backend is started after the server starts listening.
	AuditBackend audit.Backend

	SecureServingInfo *SecureServingInfo
	// numerical ports, set after listening
	effectiveSecurePort int

	postStartHookLock      sync.Mutex
	postStartHooks         map[string]postStartHookEntry
	postStartHooksCalled   bool
	disabledPostStartHooks sets.String

	// healthz checks
	healthzLock    sync.Mutex
	healthzChecks  []healthz.HealthzChecker
	healthzCreated bool
}

func (s *GenericAPIServer) ListedPaths() []string {
	return s.listedPathProvider.ListedPaths()
}

type preparedGenericAPIServer struct {
	*GenericAPIServer
}

// PrepareRun does post API installation setup steps.
func (s *GenericAPIServer) PrepareRun() preparedGenericAPIServer {
	if s.swaggerConfig != nil {
		routes.Swagger{Config: s.swaggerConfig}.Install(s.Handler.GoRestfulContainer)
	}
	if s.openAPIConfig != nil {
		routes.OpenAPI{
			Config: s.openAPIConfig,
		}.Install(s.Handler.GoRestfulContainer, s.Handler.NonGoRestfulMux)
	}

	s.installHealthz()

	return preparedGenericAPIServer{s}
}

func (s *GenericAPIServer) UnprotectedHandler() http.Handler {
	// when we delegate, we need the server we're delegating to choose whether or not to use gorestful
	return s.Handler.Director
}

type DelegationTarget interface {
	// UnprotectedHandler returns a handler that is NOT protected by a normal chain
	UnprotectedHandler() http.Handler

	// ListedPaths returns the paths for supporting an index
	ListedPaths() []string
}

// Run spawns the secure http server. It only returns if stopCh is closed
// or the secure port cannot be listened on initially.
func (s preparedGenericAPIServer) Run(stopCh <-chan struct{}) error {
	err := s.NonBlockingRun(stopCh)
	if err != nil {
		return err
	}

	<-stopCh

	if s.GenericAPIServer.AuditBackend != nil {
		s.GenericAPIServer.AuditBackend.Shutdown()
	}

	return nil
}

// NonBlockingRun spawns the secure http server. An error is
// returned if the secure port cannot be listened on.
func (s preparedGenericAPIServer) NonBlockingRun(stopCh <-chan struct{}) error {
	// Use an internal stop channel to allow cleanup of the listeners on error.
	internalStopCh := make(chan struct{})

	if s.SecureServingInfo != nil && s.Handler != nil {
		if err := s.serveSecurely(internalStopCh); err != nil {
			close(internalStopCh)
			return err
		}
	}

	// Now that listener have bound successfully, it is the
	// responsibility of the caller to close the provided channel to
	// ensure cleanup.
	go func() {
		<-stopCh
		close(internalStopCh)
	}()

	// Start the audit backend before any request comes in. This means we cannot turn it into a
	// post start hook because without calling Backend.Run the Backend.ProcessEvents call might block.
	if s.AuditBackend != nil {
		if err := s.AuditBackend.Run(stopCh); err != nil {
			return fmt.Errorf("failed to run the audit backend: %v", err)
		}
	}

	s.RunPostStartHooks(stopCh)

	if _, err := systemd.SdNotify(true, "READY=1\n"); err != nil {
		glog.Errorf("Unable to send systemd daemon successful start message: %v\n", err)
	}

	return nil
}
