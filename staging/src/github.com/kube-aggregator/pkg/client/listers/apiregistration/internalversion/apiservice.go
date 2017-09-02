package internalversion

import (
	"k8s.io/apimachinery/pkg/labels"
	apiregistration "github.com/HuZhou/kube-aggregator/pkg/apis/apiregistration"
)

// APIServiceLister helps list APIServices.
type APIServiceLister interface {
	// List lists all APIServices in the indexer.
	List(selector labels.Selector) (ret []*apiregistration.APIService, err error)
	// Get retrieves the APIService from the index for a given name.
	Get(name string) (*apiregistration.APIService, error)
	APIServiceListerExpansion
}
