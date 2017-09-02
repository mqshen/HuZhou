package filters

import (
	"github.com/golang/glog"
	"net/http"
	"k8s.io/apimachinery/pkg/util/runtime"
	"github.com/HuZhou/apiserver/pkg/server/httplog"
	"runtime/debug"
)

// WithPanicRecovery wraps an http Handler to recover and log panics.
func WithPanicRecovery(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer runtime.HandleCrash(func(err interface{}) {
			http.Error(w, "This request caused apisever to panic. Look in log for details.", http.StatusInternalServerError)
			glog.Errorf("APIServer panic'd on %v %v: %v\n%s\n", req.Method, req.RequestURI, err, debug.Stack())
		})

		logger := httplog.NewLogged(req, &w)
		defer logger.Log()

		// Dispatch to the internal handler
		handler.ServeHTTP(w, req)
	})
}