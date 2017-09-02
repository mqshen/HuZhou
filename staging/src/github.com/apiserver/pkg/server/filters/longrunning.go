package filters

import (
	"net/http"
	"k8s.io/apimachinery/pkg/util/sets"

	apirequest "github.com/HuZhou/apiserver/pkg/endpoints/request"
)

// BasicLongRunningRequestCheck returns true if the given request has one of the specified verbs or one of the specified subresources
func BasicLongRunningRequestCheck(longRunningVerbs, longRunningSubresources sets.String) apirequest.LongRunningRequestCheck {
	return func(r *http.Request, requestInfo *apirequest.RequestInfo) bool {
		if longRunningVerbs.Has(requestInfo.Verb) {
			return true
		}
		if requestInfo.IsResourceRequest && longRunningSubresources.Has(requestInfo.Subresource) {
			return true
		}
		return false
	}
}
