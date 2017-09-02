package routes

import (
	"github.com/emicklei/go-restful-swagger12"
	"github.com/emicklei/go-restful"
)

// Swagger installs the /swaggerapi/ endpoint to allow schema discovery
// and traversal. It is optional to allow consumers of the Kubernetes GenericAPIServer to
// register their own web services into the Kubernetes mux prior to initialization
// of swagger, so that other resource types show up in the documentation.
type Swagger struct {
	Config *swagger.Config
}

// Install adds the SwaggerUI webservice to the given mux.
func (s Swagger) Install(c *restful.Container) {
	s.Config.WebServices = c.RegisteredWebServices()
	swagger.RegisterSwaggerService(*s.Config, c)
}