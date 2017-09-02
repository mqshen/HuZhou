package apiserver

import (
	"k8s.io/kube-openapi/pkg/handler"
	"github.com/go-openapi/spec"
	"github.com/HuZhou/kube-aggregator/pkg/apis/apiregistration"
	"github.com/HuZhou/apiserver/pkg/endpoints/request"
)

type openAPIAggregator struct {
	// Map of API Services' OpenAPI specs by their name
	openAPISpecs map[string]*openAPISpecInfo

	// provided for dynamic OpenAPI spec
	openAPIService *handler.OpenAPIService

	// Aggregator's OpenAPI spec (holds apiregistration group).
	aggregatorOpenAPISpec *spec.Swagger

	// Local (in process) delegate's OpenAPI spec.
	inProcessDelegatesOpenAPISpec *spec.Swagger

	contextMapper request.RequestContextMapper
}

// openAPISpecInfo is used to store OpenAPI spec with its priority.
// It can be used to sort specs with their priorities.
type openAPISpecInfo struct {
	apiService apiregistration.APIService
	spec       *spec.Swagger
}