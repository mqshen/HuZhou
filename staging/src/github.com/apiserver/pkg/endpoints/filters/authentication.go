package filters

import (
	"github.com/golang/glog"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	genericapirequest "github.com/HuZhou/apiserver/pkg/endpoints/request"
	"github.com/HuZhou/apiserver/pkg/authentication/authenticator"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
	"k8s.io/apimachinery/pkg/runtime"
	"errors"
	"github.com/HuZhou/apiserver/pkg/endpoints/request"
	"github.com/HuZhou/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apimachinery/pkg/runtime/schema"
)


var (
	authenticatedUserCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authenticated_user_requests",
			Help: "Counter of authenticated requests broken out by username.",
		},
		[]string{"username"},
	)
)

// WithAuthentication creates an http handler that tries to authenticate the given request as a user, and then
// stores any such user found onto the provided context for the request. If authentication fails or returns an error
// the failed handler is used. On success, "Authorization" header is removed from the request and handler
// is invoked to serve the request.
func WithAuthentication(handler http.Handler, mapper genericapirequest.RequestContextMapper, auth authenticator.Request, failed http.Handler) http.Handler {
	if auth == nil {
		glog.Warningf("Authentication is disabled")
		return handler
	}
	return genericapirequest.WithRequestContext(
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			user, ok, err := auth.AuthenticateRequest(req)
			if err != nil || !ok {
				if err != nil {
					glog.Errorf("Unable to authenticate the request due to an error: %v", err)
				}
				failed.ServeHTTP(w, req)
				return
			}

			// authorization header is not required anymore in case of a successful authentication.
			req.Header.Del("Authorization")

			if ctx, ok := mapper.Get(req); ok {
				mapper.Update(req, genericapirequest.WithUser(ctx, user))
			}

			authenticatedUserCounter.WithLabelValues(compressUsername(user.GetName())).Inc()

			handler.ServeHTTP(w, req)
		}),
		mapper,
	)
}

func compressUsername(username string) string {
	switch {
	// Known internal identities.
	case username == "admin" ||
		username == "client" ||
		username == "kube_proxy" ||
		username == "kubelet" ||
		username == "system:serviceaccount:kube-system:default":
		return username
		// Probably an email address.
	case strings.Contains(username, "@"):
		return "email_id"
		// Anything else (custom service accounts, custom external identities, etc.)
	default:
		return "other"
	}
}

func Unauthorized(requestContextMapper request.RequestContextMapper, s runtime.NegotiatedSerializer, supportsBasicAuth bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if supportsBasicAuth {
			w.Header().Set("WWW-Authenticate", `Basic realm="kubernetes-master"`)
		}
		ctx, ok := requestContextMapper.Get(req)
		if !ok {
			responsewriters.InternalError(w, req, errors.New("no context found for request"))
			return
		}
		requestInfo, found := request.RequestInfoFrom(ctx)
		if !found {
			responsewriters.InternalError(w, req, errors.New("no RequestInfo found in the context"))
			return
		}

		gv := schema.GroupVersion{Group: requestInfo.APIGroup, Version: requestInfo.APIVersion}
		responsewriters.ErrorNegotiated(ctx, apierrors.NewUnauthorized("Unauthorized"), s, gv, w, req)
	})
}