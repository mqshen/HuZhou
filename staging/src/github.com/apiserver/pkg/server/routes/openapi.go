package routes

import (

	"github.com/HuZhou/apiserver/pkg/server/mux"
	"github.com/emicklei/go-restful"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/handler"
	"github.com/golang/glog"
)

// OpenAPI installs spec endpoints for each web service.
type OpenAPI struct {
	Config *common.Config
}


// Install adds the SwaggerUI webservice to the given mux.
func (oa OpenAPI) Install(c *restful.Container, mux *mux.PathRecorderMux) {
	_, err := handler.BuildAndRegisterOpenAPIService("/swagger.json", c.RegisteredWebServices(), oa.Config, mux)
	if err != nil {
		glog.Fatalf("Failed to register open api spec for root: %v", err)
	}
}