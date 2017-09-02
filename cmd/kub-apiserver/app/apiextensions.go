package app

import (
	genericapiserver "github.com/HuZhou/apiserver/pkg/server"
	genericoptions "github.com/HuZhou/apiserver/pkg/server/options"
	apiextensionscmd "github.com/HuZhou/apiextensions-apiserver/pkg/cmd/server"
	apiextensionsapiserver "github.com/HuZhou/apiextensions-apiserver/pkg/apiserver"

	"github.com/mqshen/HuZhou/cmd/kub-apiserver/app/options"
	"github.com/HuZhou/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)
func createAPIExtensionsConfig(kubeAPIServerConfig genericapiserver.Config, commandOptions *options.ServerRunOptions) (*apiextensionsapiserver.Config, error) {
	// make a shallow copy to let us twiddle a few things
	// most of the config actually remains the same.  We only need to mess with a couple items related to the particulars of the apiextensions
	genericConfig := kubeAPIServerConfig

	// copy the etcd options so we don't mutate originals.
	etcdOptions := *commandOptions.Etcd
	etcdOptions.StorageConfig.Codec = apiextensionsapiserver.Codecs.LegacyCodec(v1beta1.SchemeGroupVersion)
	etcdOptions.StorageConfig.Copier = apiextensionsapiserver.Scheme
	genericConfig.RESTOptionsGetter = &genericoptions.SimpleRestOptionsFactory{Options: etcdOptions}

	apiextensionsConfig := &apiextensionsapiserver.Config{
		GenericConfig:        &genericConfig,
		CRDRESTOptionsGetter: apiextensionscmd.NewCRDRESTOptionsGetter(etcdOptions),
	}

	return apiextensionsConfig, nil

}

func createAPIExtensionsServer(apiextensionsConfig *apiextensionsapiserver.Config, delegateAPIServer genericapiserver.DelegationTarget) (*apiextensionsapiserver.CustomResourceDefinitions, error) {
	apiextensionsServer, err := apiextensionsConfig.Complete().New(delegateAPIServer)
	if err != nil {
		return nil, err
	}

	return apiextensionsServer, nil
}