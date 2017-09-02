package apiserver

import (
	genericapiserver "github.com/HuZhou/apiserver/pkg/server"
	genericregistry "github.com/HuZhou/apiserver/pkg/registry/generic"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
)

var (
	Scheme               = runtime.NewScheme()
	Codecs               = serializer.NewCodecFactory(Scheme)
)
type Config struct {
	GenericConfig *genericapiserver.Config

	CRDRESTOptionsGetter genericregistry.RESTOptionsGetter
}

type CustomResourceDefinitions struct {
	GenericAPIServer *genericapiserver.GenericAPIServer

	// provided for easier embedding
	//Informers internalinformers.SharedInformerFactory
}

func (c *Config) Complete() completedConfig {
	c.GenericConfig.EnableDiscovery = false
	c.GenericConfig.Complete()

	c.GenericConfig.Version = &version.Info{
		Major: "0",
		Minor: "1",
	}

	return completedConfig{c}
}

type completedConfig struct {
	*Config
}


func (c completedConfig) New(delegationTarget genericapiserver.DelegationTarget) (*CustomResourceDefinitions, error) {
	genericServer, err := c.Config.GenericConfig.SkipComplete().New("apiextensions-apiserver", delegationTarget) // completion is done in Complete, no need for a second time
	if err != nil {
		return nil, err
	}

	s := &CustomResourceDefinitions{
		GenericAPIServer: genericServer,
	}
	return s, nil
}