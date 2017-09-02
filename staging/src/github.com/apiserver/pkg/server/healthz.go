package server

import (
	"fmt"
	"github.com/HuZhou/apiserver/pkg/server/healthz"
)

func (s *GenericAPIServer) AddHealthzChecks(checks ...healthz.HealthzChecker) error {
	s.healthzLock.Lock()
	defer s.healthzLock.Unlock()

	if s.healthzCreated {
		return fmt.Errorf("unable to add because the healthz endpoint has already been created")
	}

	s.healthzChecks = append(s.healthzChecks, checks...)
	return nil
}

// installHealthz creates the healthz endpoint for this server
func (s *GenericAPIServer) installHealthz() {
	s.healthzLock.Lock()
	defer s.healthzLock.Unlock()
	s.healthzCreated = true

	healthz.InstallHandler(s.Handler.NonGoRestfulMux, s.healthzChecks...)
}