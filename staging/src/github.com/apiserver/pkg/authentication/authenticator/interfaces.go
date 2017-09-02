package authenticator

import (
	"net/http"
	"github.com/HuZhou/apiserver/pkg/authentication/user"
)

// Request attempts to extract authentication information from a request and returns
// information about the current user and true if successful, false if not successful,
// or an error if the request could not be checked.
type Request interface {
	AuthenticateRequest(req *http.Request) (user.Info, bool, error)
}