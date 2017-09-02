package healthz

import (
	"net/http"
	"fmt"
	"bytes"
)


// PingHealthz returns true automatically when checked
var PingHealthz HealthzChecker = ping{}

// ping implements the simplest possible healthz checker.
type ping struct{}

func (ping) Name() string {
	return "ping"
}

// PingHealthz is a health check that returns true.
func (ping) Check(_ *http.Request) error {
	return nil
}
// HealthzChecker is a named healthz checker.
type HealthzChecker interface {
	Name() string
	Check(req *http.Request) error
}

// InstallHandler registers a handler for health checking on the path "/healthz" to mux.
func InstallHandler(mux mux, checks ...HealthzChecker) {
	if len(checks) == 0 {
		checks = []HealthzChecker{PingHealthz}
	}
	mux.Handle("/healthz", handleRootHealthz(checks...))
	for _, check := range checks {
		mux.Handle(fmt.Sprintf("/healthz/%v", check.Name()), adaptCheckToHandler(check.Check))
	}
}

// handleRootHealthz returns an http.HandlerFunc that serves the provided checks.
func handleRootHealthz(checks ...HealthzChecker) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failed := false
		var verboseOut bytes.Buffer
		for _, check := range checks {
			if check.Check(r) != nil {
				// don't include the error since this endpoint is public.  If someone wants more detail
				// they should have explicit permission to the detailed checks.
				fmt.Fprintf(&verboseOut, "[-]%v failed: reason withheld\n", check.Name())
				failed = true
			} else {
				fmt.Fprintf(&verboseOut, "[+]%v ok\n", check.Name())
			}
		}
		// always be verbose on failure
		if failed {
			http.Error(w, fmt.Sprintf("%vhealthz check failed", verboseOut.String()), http.StatusInternalServerError)
			return
		}

		if _, found := r.URL.Query()["verbose"]; !found {
			fmt.Fprint(w, "ok")
			return
		}

		verboseOut.WriteTo(w)
		fmt.Fprint(w, "healthz check passed\n")
	})
}

// mux is an interface describing the methods InstallHandler requires.
type mux interface {
	Handle(pattern string, handler http.Handler)
}

// adaptCheckToHandler returns an http.HandlerFunc that serves the provided checks.
func adaptCheckToHandler(c func(r *http.Request) error) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := c(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("internal server error: %v", err), http.StatusInternalServerError)
		} else {
			fmt.Fprint(w, "ok")
		}
	})
}