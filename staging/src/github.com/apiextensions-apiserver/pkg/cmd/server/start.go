package server

import (
	"github.com/HuZhou/apiextensions-apiserver/pkg/apiserver"
	genericoptions "github.com/HuZhou/apiserver/pkg/server/options"
	genericregistry "github.com/HuZhou/apiserver/pkg/registry/generic"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)
func NewCRDRESTOptionsGetter(etcdOptions genericoptions.EtcdOptions) genericregistry.RESTOptionsGetter {
	ret := apiserver.CRDRESTOptionsGetter{
		StorageConfig:           etcdOptions.StorageConfig,
		StoragePrefix:           etcdOptions.StorageConfig.Prefix,
		EnableWatchCache:        etcdOptions.EnableWatchCache,
		DefaultWatchCacheSize:   etcdOptions.DefaultWatchCacheSize,
		EnableGarbageCollection: etcdOptions.EnableGarbageCollection,
		DeleteCollectionWorkers: etcdOptions.DeleteCollectionWorkers,
	}
	ret.StorageConfig.Codec = unstructured.UnstructuredJSONScheme
	ret.StorageConfig.Copier = apiserver.UnstructuredCopier{}

	return ret
}
