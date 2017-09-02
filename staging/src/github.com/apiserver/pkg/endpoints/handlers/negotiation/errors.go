package negotiation

import (

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"fmt"
	"strings"
	"net/http"
)

// errNotAcceptable indicates Accept negotiation has failed
type errNotAcceptable struct {
	accepted []string
}

func (e errNotAcceptable) Error() string {
	return fmt.Sprintf("only the following media types are accepted: %v", strings.Join(e.accepted, ", "))
}

func (e errNotAcceptable) Status() metav1.Status {
	return metav1.Status{
		Status:  metav1.StatusFailure,
		Code:    http.StatusNotAcceptable,
		Reason:  metav1.StatusReason("NotAcceptable"),
		Message: e.Error(),
	}
}

func NewNotAcceptableError(accepted []string) error {
	return errNotAcceptable{accepted}
}
