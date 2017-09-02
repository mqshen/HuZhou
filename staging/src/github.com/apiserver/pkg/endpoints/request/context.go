package request

import (
	stderrs "errors"
	"time"

	"golang.org/x/net/context"
	"github.com/HuZhou/apiserver/pkg/authentication/user"
	"github.com/HuZhou/apiserver/pkg/apis/audit"
)

// The key type is unexported to prevent collisions
type key int

const (
	// namespaceKey is the context key for the request namespace.
	namespaceKey key = iota

	// userKey is the context key for the request user.
	userKey

	// uidKey is the context key for the uid to assign to an object on create.
	uidKey

	// userAgentKey is the context key for the request user agent.
	userAgentKey

	// auditKey is the context key for the audit event.
	auditKey

	namespaceDefault = "default" // TODO(sttts): solve import cycle when using metav1.NamespaceDefault
)

type Context interface {
	// Value returns the value associated with key or nil if none.
	Value(key interface{}) interface{}

	// Deadline returns the time when this Context will be canceled, if any.
	Deadline() (deadline time.Time, ok bool)

	// Done returns a channel that is closed when this Context is canceled
	// or times out.
	Done() <-chan struct{}

	// Err indicates why this context was canceled, after the Done channel
	// is closed.
	Err() error
}

// WithValue returns a copy of parent in which the value associated with key is val.
func WithValue(parent Context, key interface{}, val interface{}) Context {
	internalCtx, ok := parent.(context.Context)
	if !ok {
		panic(stderrs.New("Invalid context type"))
	}
	return context.WithValue(internalCtx, key, val)
}

// UserFrom returns the value of the user key on the ctx
func UserFrom(ctx Context) (user.Info, bool) {
	user, ok := ctx.Value(userKey).(user.Info)
	return user, ok
}

// WithAuditEvent returns set audit event struct.
func WithAuditEvent(parent Context, ev *audit.Event) Context {
	return WithValue(parent, auditKey, ev)
}


// WithUser returns a copy of parent in which the user value is set
func WithUser(parent Context, user user.Info) Context {
	return WithValue(parent, userKey, user)
}

// AuditEventFrom returns the audit event struct on the ctx
func AuditEventFrom(ctx Context) *audit.Event {
	ev, _ := ctx.Value(auditKey).(*audit.Event)
	return ev
}