package audit

import (
	"github.com/pborman/uuid"
	"k8s.io/apimachinery/pkg/types"
	"net/http"

	auditinternal "github.com/HuZhou/apiserver/pkg/apis/audit"
	authenticationv1 "github.com/HuZhou/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"github.com/HuZhou/apiserver/pkg/authorization/authorizer"
	"time"
	"strings"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"bytes"
	"fmt"
)

func NewEventFromRequest(req *http.Request, level auditinternal.Level, attribs authorizer.Attributes) (*auditinternal.Event, error) {
	ev := &auditinternal.Event{
		Timestamp:  metav1.NewTime(time.Now()),
		Verb:       attribs.GetVerb(),
		RequestURI: req.URL.RequestURI(),
	}

	ev.Level = level

	// prefer the id from the headers. If not available, create a new one.
	// TODO(audit): do we want to forbid the header for non-front-proxy users?
	ids := req.Header.Get(auditinternal.HeaderAuditID)
	if ids != "" {
		ev.AuditID = types.UID(ids)
	} else {
		ev.AuditID = types.UID(uuid.NewRandom().String())
	}

	ips := utilnet.SourceIPs(req)
	ev.SourceIPs = make([]string, len(ips))
	for i := range ips {
		ev.SourceIPs[i] = ips[i].String()
	}

	if user := attribs.GetUser(); user != nil {
		ev.User.Username = user.GetName()
		ev.User.Extra = map[string]auditinternal.ExtraValue{}
		for k, v := range user.GetExtra() {
			ev.User.Extra[k] = auditinternal.ExtraValue(v)
		}
		ev.User.Groups = user.GetGroups()
		ev.User.UID = user.GetUID()
	}

	if asuser := req.Header.Get(authenticationv1.ImpersonateUserHeader); len(asuser) > 0 {
		ev.ImpersonatedUser = &auditinternal.UserInfo{
			Username: asuser,
		}
		if requestedGroups := req.Header[authenticationv1.ImpersonateGroupHeader]; len(requestedGroups) > 0 {
			ev.ImpersonatedUser.Groups = requestedGroups
		}

		ev.ImpersonatedUser.Extra = map[string]auditinternal.ExtraValue{}
		for k, v := range req.Header {
			if !strings.HasPrefix(k, authenticationv1.ImpersonateUserExtraHeaderPrefix) {
				continue
			}
			k = k[len(authenticationv1.ImpersonateUserExtraHeaderPrefix):]
			ev.ImpersonatedUser.Extra[k] = auditinternal.ExtraValue(v)
		}
	}

	if attribs.IsResourceRequest() {
		ev.ObjectRef = &auditinternal.ObjectReference{
			Namespace:   attribs.GetNamespace(),
			Name:        attribs.GetName(),
			Resource:    attribs.GetResource(),
			Subresource: attribs.GetSubresource(),
			APIVersion:  attribs.GetAPIGroup() + "/" + attribs.GetAPIVersion(),
		}
	}

	return ev, nil
}

// LogResponseObject fills in the response object into an audit event. The passed runtime.Object
// will be converted to the given gv.
func LogResponseObject(ae *auditinternal.Event, obj runtime.Object, gv schema.GroupVersion, s runtime.NegotiatedSerializer) {
	if ae == nil || ae.Level.Less(auditinternal.LevelMetadata) {
		return
	}
	if status, ok := obj.(*metav1.Status); ok {
		ae.ResponseStatus = status
	}

	if ae.Level.Less(auditinternal.LevelRequestResponse) {
		return
	}
	// TODO(audit): hook into the serializer to avoid double conversion
	var err error
	ae.ResponseObject, err = encodeObject(obj, gv, s)
	if err != nil {
		glog.Warningf("Audit failed for %q response: %v", reflect.TypeOf(obj).Name(), err)
	}
}

func encodeObject(obj runtime.Object, gv schema.GroupVersion, serializer runtime.NegotiatedSerializer) (*runtime.Unknown, error) {
	supported := serializer.SupportedMediaTypes()
	for i := range supported {
		if supported[i].MediaType == "application/json" {
			enc := serializer.EncoderForVersion(supported[i].Serializer, gv)
			var buf bytes.Buffer
			if err := enc.Encode(obj, &buf); err != nil {
				return nil, fmt.Errorf("encoding failed: %v", err)
			}

			return &runtime.Unknown{
				Raw:         buf.Bytes(),
				ContentType: runtime.ContentTypeJSON,
			}, nil
		}
	}
	return nil, fmt.Errorf("no json encoder found")
}