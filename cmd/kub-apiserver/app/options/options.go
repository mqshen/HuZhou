package options

import (
	genericoptions "github.com/HuZhou/apiserver/pkg/server/options"
	kubeoptions "github.com/mqshen/HuZhou/pkg/kubeapiserver/options"
	"github.com/spf13/pflag"
	"github.com/HuZhou/apiserver/pkg/storage/storagebackend"
	"github.com/mqshen/HuZhou/pkg/api"
)

type ServerRunOptions struct {
	Etcd                    *genericoptions.EtcdOptions
	InsecureServing         *kubeoptions.InsecureServingOptions
	SSHUser                 string
}

func NewServerRunOptions() *ServerRunOptions {
	s := ServerRunOptions{
		Etcd:                 genericoptions.NewEtcdOptions(storagebackend.NewDefaultConfig(kubeoptions.DefaultEtcdPathPrefix, api.Scheme, nil)),
		InsecureServing:      kubeoptions.NewInsecureServingOptions(),
	}
	return &s
}

func (s *ServerRunOptions) AddFlags(fs *pflag.FlagSet) {

}