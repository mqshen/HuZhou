package crdregistration

import (
	"k8s.io/client-go/util/workqueue"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type crdRegistrationController struct {


	syncHandler func(groupVersion schema.GroupVersion) error

	// queue is where incoming work is placed to de-dup and to allow "easy" rate limited requeues on errors
	// this is actually keyed by a groupVersion
	queue workqueue.RateLimitingInterface
}

func NewAutoRegistrationController() *crdRegistrationController{
	c := &crdRegistrationController{
		queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "crd-autoregister"),
	}
	return c
}
