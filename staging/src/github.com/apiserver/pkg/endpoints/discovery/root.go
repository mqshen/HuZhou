package discovery

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/emicklei/go-restful"
)

// GroupManager is an interface that allows dynamic mutation of the existing webservice to handle
// API groups being added or removed.
type GroupManager interface {
	AddGroup(apiGroup metav1.APIGroup)
	RemoveGroup(groupName string)

	WebService() *restful.WebService
}

