package server

import (
	"fmt"
	"github.com/HuZhou/apiserver/pkg/server/healthz"
	"errors"
	"net/http"
	restclient "k8s.io/client-go/rest"
	"github.com/golang/glog"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

type PostStartHookFunc func(context PostStartHookContext) error

type PostStartHookContext struct {
	// LoopbackClientConfig is a config for a privileged loopback connection to the API server
	LoopbackClientConfig *restclient.Config
	// StopCh is the channel that will be closed when the server stops
	StopCh <-chan struct{}
}

// AddPostStartHook allows you to add a PostStartHook.
func (s *GenericAPIServer) AddPostStartHook(name string, hook PostStartHookFunc) error {
	if len(name) == 0 {
		return fmt.Errorf("missing name")
	}
	if hook == nil {
		return nil
	}
	if s.disabledPostStartHooks.Has(name) {
		return nil
	}

	s.postStartHookLock.Lock()
	defer s.postStartHookLock.Unlock()

	if s.postStartHooksCalled {
		return fmt.Errorf("unable to add %q because PostStartHooks have already been called", name)
	}
	if _, exists := s.postStartHooks[name]; exists {
		return fmt.Errorf("unable to add %q because it is already registered", name)
	}

	// done is closed when the poststarthook is finished.  This is used by the health check to be able to indicate
	// that the poststarthook is finished
	done := make(chan struct{})
	s.AddHealthzChecks(postStartHookHealthz{name: "poststarthook/" + name, done: done})
	s.postStartHooks[name] = postStartHookEntry{hook: hook, done: done}

	return nil
}

// postStartHookHealthz implements a healthz check for poststarthooks.  It will return a "hookNotFinished"
// error until the poststarthook is finished.
type postStartHookHealthz struct {
	name string

	// done will be closed when the postStartHook is finished
	done chan struct{}
}

var _ healthz.HealthzChecker = postStartHookHealthz{}

func (h postStartHookHealthz) Name() string {
	return h.name
}

var hookNotFinished = errors.New("not finished")

func (h postStartHookHealthz) Check(req *http.Request) error {
	select {
	case <-h.done:
		return nil
	default:
		return hookNotFinished
	}
}

// RunPostStartHooks runs the PostStartHooks for the server
func (s *GenericAPIServer) RunPostStartHooks(stopCh <-chan struct{}) {
	s.postStartHookLock.Lock()
	defer s.postStartHookLock.Unlock()
	s.postStartHooksCalled = true

	context := PostStartHookContext{
		LoopbackClientConfig: s.LoopbackClientConfig,
		StopCh:               stopCh,
	}

	for hookName, hookEntry := range s.postStartHooks {
		go runPostStartHook(hookName, hookEntry, context)
	}
}


func runPostStartHook(name string, entry postStartHookEntry, context PostStartHookContext) {
	var err error
	func() {
		// don't let the hook *accidentally* panic and kill the server
		defer utilruntime.HandleCrash()
		err = entry.hook(context)
	}()
	// if the hook intentionally wants to kill server, let it.
	if err != nil {
		glog.Fatalf("PostStartHook %q failed: %v", name, err)
	}
	close(entry.done)
}
