package filters

import (
	"fmt"
	"net/http"


	apirequest "github.com/HuZhou/apiserver/pkg/endpoints/request"
	genericapirequest "github.com/HuZhou/apiserver/pkg/endpoints/request"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/sets"
	"github.com/HuZhou/apiserver/pkg/authentication/user"
	"github.com/HuZhou/apiserver/pkg/endpoints/metrics"
	"strings"
	"time"
)

// Constant for the retry-after interval on rate limiting.
// TODO: maybe make this dynamic? or user-adjustable?
const retryAfter = "1"

var nonMutatingRequestVerbs = sets.NewString("get", "list", "watch")

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, "Internal Server Error: %#v", r.RequestURI)
	glog.Errorf(err.Error())
}

// WithMaxInFlightLimit limits the number of in-flight requests to buffer size of the passed in channel.
func WithMaxInFlightLimit(
	handler http.Handler,
	nonMutatingLimit int,
	mutatingLimit int,
	requestContextMapper genericapirequest.RequestContextMapper,
	longRunningRequestCheck apirequest.LongRunningRequestCheck,
) http.Handler {
	if nonMutatingLimit == 0 && mutatingLimit == 0 {
		return handler
	}
	var nonMutatingChan chan bool
	var mutatingChan chan bool
	if nonMutatingLimit != 0 {
		nonMutatingChan = make(chan bool, nonMutatingLimit)
	}
	if mutatingLimit != 0 {
		mutatingChan = make(chan bool, mutatingLimit)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := requestContextMapper.Get(r)
		if !ok {
			handleError(w, r, fmt.Errorf("no context found for request, handler chain must be wrong"))
			return
		}
		requestInfo, ok := apirequest.RequestInfoFrom(ctx)
		if !ok {
			handleError(w, r, fmt.Errorf("no RequestInfo found in context, handler chain must be wrong"))
			return
		}

		// Skip tracking long running events.
		if longRunningRequestCheck != nil && longRunningRequestCheck(r, requestInfo) {
			handler.ServeHTTP(w, r)
			return
		}

		var c chan bool
		if !nonMutatingRequestVerbs.Has(requestInfo.Verb) {
			c = mutatingChan
		} else {
			c = nonMutatingChan
		}

		if c == nil {
			handler.ServeHTTP(w, r)
		} else {

			select {
			case c <- true:
				defer func() { <-c }()
				handler.ServeHTTP(w, r)

			default:
				// at this point we're about to return a 429, BUT not all actors should be rate limited.  A system:master is so powerful
				// that he should always get an answer.  It's a super-admin or a loopback connection.
				if currUser, ok := apirequest.UserFrom(ctx); ok {
					for _, group := range currUser.GetGroups() {
						if group == user.SystemPrivilegedGroup {
							handler.ServeHTTP(w, r)
							return
						}
					}
				}
				scope := "cluster"
				if requestInfo.Namespace != "" {
					scope = "namespace"
				}
				if requestInfo.IsResourceRequest {
					metrics.MonitorRequest(r, strings.ToUpper(requestInfo.Verb), requestInfo.Resource, requestInfo.Subresource, "", scope, http.StatusTooManyRequests, 0, time.Now())
				} else {
					metrics.MonitorRequest(r, strings.ToUpper(requestInfo.Verb), "", requestInfo.Path, "", scope, http.StatusTooManyRequests, 0, time.Now())
				}
				tooManyRequests(r, w)
			}
		}
	})
}

func tooManyRequests(req *http.Request, w http.ResponseWriter) {
	// Return a 429 status indicating "Too Many Requests"
	w.Header().Set("Retry-After", retryAfter)
	http.Error(w, "Too many requests, please try again later.", http.StatusTooManyRequests)
}