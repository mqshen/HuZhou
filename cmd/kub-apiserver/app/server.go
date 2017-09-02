package app

import (
	"github.com/golang/glog"
	"github.com/mqshen/HuZhou/cmd/kub-apiserver/app/options"
	"github.com/mqshen/HuZhou/pkg/version"

	"github.com/mqshen/HuZhou/pkg/api"
	genericapiserver "github.com/HuZhou/apiserver/pkg/server"
	"github.com/mqshen/HuZhou/pkg/master"
	"net/http"
	"github.com/mqshen/HuZhou/pkg/master/tunneler"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	kubeserver "github.com/mqshen/HuZhou/pkg/kubeapiserver/server"
	"crypto/tls"
	"fmt"
)

// Run runs the specified APIServer.  This should never exit.
func Run(runOptions *options.ServerRunOptions, stopCh <-chan struct{}) error {
	// To help debugging, immediately log version
	glog.Infof("ttttt Version: %+v", version.Get())

	server, err := CreateServerChain(runOptions, stopCh)
	if err != nil {
		return err
	}

	return server.PrepareRun().Run(stopCh)
}

func CreateServerChain(runOptions *options.ServerRunOptions, stopCh <-chan struct{}) (*genericapiserver.GenericAPIServer, error) {
	nodeTunneler, proxyTransport, err := CreateNodeDialer(runOptions)
	if err != nil {
		return nil, err
	}

	kubeAPIServerConfig, insecureServingOptions, err := CreateKubeAPIServerConfig(runOptions, nodeTunneler, proxyTransport)
	if err != nil {
		return nil, err
	}
	// TPRs are enabled and not yet beta, since this these are the successor, they fall under the same enablement rule
	// If additional API servers are added, they should be gated.
	apiExtensionsConfig, err := createAPIExtensionsConfig(*kubeAPIServerConfig.GenericConfig, runOptions)
	if err != nil {
		return nil, err
	}
	apiExtensionsServer, err := createAPIExtensionsServer(apiExtensionsConfig, genericapiserver.EmptyDelegate)
	if err != nil {
		return nil, err
	}

	kubeAPIServer, err := CreateKubeAPIServer(kubeAPIServerConfig, apiExtensionsServer.GenericAPIServer)

	// aggregator comes last in the chain
	aggregatorConfig, err := createAggregatorConfig(*kubeAPIServerConfig.GenericConfig, runOptions, proxyTransport)
	if err != nil {
		return nil, err
	}
	aggregatorServer, err := createAggregatorServer(aggregatorConfig, kubeAPIServer.GenericAPIServer)
	if err != nil {
		return nil, err
	}

	insecureHandlerChain := kubeserver.BuildInsecureHandlerChain(aggregatorServer.GenericAPIServer.UnprotectedHandler(), kubeAPIServerConfig.GenericConfig)

	fmt.Println(insecureServingOptions)//.UnprotectedHandler()
	fmt.Println(insecureHandlerChain)

	if err := kubeserver.NonBlockingRun(insecureServingOptions, insecureHandlerChain, stopCh); err != nil {
		return nil, err
	}

	return kubeAPIServer.GenericAPIServer, nil

}

func CreateKubeAPIServerConfig(s *options.ServerRunOptions, nodeTunneler tunneler.Tunneler, proxyTransport http.RoundTripper) (*master.Config, *kubeserver.InsecureServingInfo, error) {
	genericConfig, insecureServingOptions, err := BuildGenericConfig(s)
	if err != nil {
		return nil, nil, err
	}
	config := &master.Config{
		GenericConfig: genericConfig,
	}
	return config, insecureServingOptions, nil
}

func BuildGenericConfig(s *options.ServerRunOptions) (*genericapiserver.Config, *kubeserver.InsecureServingInfo, error) {
	genericConfig := genericapiserver.NewConfig(api.Codecs)
	insecureServingOptions, err := s.InsecureServing.ApplyTo(genericConfig)
	if err != nil {
		return nil, nil, err
	}
	return genericConfig, insecureServingOptions, nil
}

// CreateNodeDialer creates the dialer infrastructure to connect to the nodes.
func CreateNodeDialer(s *options.ServerRunOptions) (tunneler.Tunneler, *http.Transport, error) {

	var nodeTunneler tunneler.Tunneler
	var proxyDialerFn utilnet.DialFunc

	if len(s.SSHUser) > 0 {

	}
	proxyTLSClientConfig := &tls.Config{InsecureSkipVerify: true}
	proxyTransport := utilnet.SetTransportDefaults(&http.Transport{
		Dial:            proxyDialerFn,
		TLSClientConfig: proxyTLSClientConfig,
	})
	return nodeTunneler, proxyTransport, nil

}

// CreateKubeAPIServer creates and wires a workable kube-apiserver
func CreateKubeAPIServer(kubeAPIServerConfig *master.Config, delegateAPIServer genericapiserver.DelegationTarget) (*master.Master, error) {
	kubeAPIServer, err := kubeAPIServerConfig.Complete().New(delegateAPIServer)
	if err != nil {
		return nil, err
	}
	kubeAPIServer.GenericAPIServer.AddPostStartHook("start-kube-apiserver-informers", func(context genericapiserver.PostStartHookContext) error {
		//sharedInformers.Start(context.StopCh)
		return nil
	})

	return kubeAPIServer, nil
}