package responsewriters

import (
	"fmt"
	"net/http"
	"strings"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"github.com/HuZhou/apiserver/pkg/endpoints/request"
	"github.com/HuZhou/apiserver/pkg/authorization/authorizer"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Avoid emitting errors that look like valid HTML. Quotes are okay.
var sanitizer = strings.NewReplacer(`&`, "&amp;", `<`, "&lt;", `>`, "&gt;")

// InternalError renders a simple internal error
func InternalError(w http.ResponseWriter, req *http.Request, err error) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, "Internal Server Error: %q: %v", sanitizer.Replace(req.RequestURI), err)
	utilruntime.HandleError(err)
}

// Forbidden renders a simple forbidden error
func Forbidden(ctx request.Context, attributes authorizer.Attributes, w http.ResponseWriter, req *http.Request, reason string, s runtime.NegotiatedSerializer) {
	msg := sanitizer.Replace(forbiddenMessage(attributes))
	w.Header().Set("X-Content-Type-Options", "nosniff")

	var errMsg string
	if len(reason) == 0 {
		errMsg = fmt.Sprintf("%s", msg)
	} else {
		errMsg = fmt.Sprintf("%s: %q", msg, reason)
	}
	gv := schema.GroupVersion{Group: attributes.GetAPIGroup(), Version: attributes.GetAPIVersion()}
	gr := schema.GroupResource{Group: attributes.GetAPIGroup(), Resource: attributes.GetResource()}
	ErrorNegotiated(ctx, apierrors.NewForbidden(gr, attributes.GetName(), fmt.Errorf(errMsg)), s, gv, w, req)
}


func forbiddenMessage(attributes authorizer.Attributes) string {
	username := ""
	if user := attributes.GetUser(); user != nil {
		username = user.GetName()
	}

	if !attributes.IsResourceRequest() {
		return fmt.Sprintf("User %q cannot %s path %q.", username, attributes.GetVerb(), attributes.GetPath())
	}

	resource := attributes.GetResource()
	if group := attributes.GetAPIGroup(); len(group) > 0 {
		resource = resource + "." + group
	}
	if subresource := attributes.GetSubresource(); len(subresource) > 0 {
		resource = resource + "/" + subresource
	}

	if ns := attributes.GetNamespace(); len(ns) > 0 {
		return fmt.Sprintf("User %q cannot %s %s in the namespace %q.", username, attributes.GetVerb(), resource, ns)
	}

	return fmt.Sprintf("User %q cannot %s %s at the cluster scope.", username, attributes.GetVerb(), resource)
}