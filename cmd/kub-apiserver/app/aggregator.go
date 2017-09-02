package app

import (

	genericapiserver "github.com/HuZhou/apiserver/pkg/server"
	aggregatorapiserver "github.com/HuZhou/kube-aggregator/pkg/apiserver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/HuZhou/kube-aggregator/pkg/apis/apiregistration"

	"github.com/mqshen/HuZhou/cmd/kub-apiserver/app/options"
	"net/http"
	"github.com/HuZhou/kube-aggregator/pkg/controllers/autoregister"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strings"
)
type priority struct {
	group   int32
	version int32
}

var apiVersionPriorities = map[schema.GroupVersion]priority{
	{Group: "", Version: "v1"}: {group: 18000, version: 1},
	// extensions is above the rest for CLI compatibility, though the level of unqalified resource compatibility we
	// can reasonably expect seems questionable.
	{Group: "extensions", Version: "v1beta1"}: {group: 17900, version: 1},
	// to my knowledge, nothing below here collides
	{Group: "apps", Version: "v1beta1"}:                          {group: 17800, version: 1},
	{Group: "apps", Version: "v1beta2"}:                          {group: 17800, version: 1},
	{Group: "authentication.k8s.io", Version: "v1"}:              {group: 17700, version: 15},
	{Group: "authentication.k8s.io", Version: "v1beta1"}:         {group: 17700, version: 9},
	{Group: "authorization.k8s.io", Version: "v1"}:               {group: 17600, version: 15},
	{Group: "authorization.k8s.io", Version: "v1beta1"}:          {group: 17600, version: 9},
	{Group: "autoscaling", Version: "v1"}:                        {group: 17500, version: 15},
	{Group: "autoscaling", Version: "v2alpha1"}:                  {group: 17500, version: 9},
	{Group: "batch", Version: "v1"}:                              {group: 17400, version: 15},
	{Group: "batch", Version: "v1beta1"}:                         {group: 17400, version: 9},
	{Group: "batch", Version: "v2alpha1"}:                        {group: 17400, version: 9},
	{Group: "certificates.k8s.io", Version: "v1beta1"}:           {group: 17300, version: 9},
	{Group: "networking.k8s.io", Version: "v1"}:                  {group: 17200, version: 15},
	{Group: "policy", Version: "v1beta1"}:                        {group: 17100, version: 9},
	{Group: "rbac.authorization.k8s.io", Version: "v1"}:          {group: 17000, version: 15},
	{Group: "rbac.authorization.k8s.io", Version: "v1beta1"}:     {group: 17000, version: 12},
	{Group: "rbac.authorization.k8s.io", Version: "v1alpha1"}:    {group: 17000, version: 9},
	{Group: "settings.k8s.io", Version: "v1alpha1"}:              {group: 16900, version: 9},
	{Group: "storage.k8s.io", Version: "v1"}:                     {group: 16800, version: 15},
	{Group: "storage.k8s.io", Version: "v1beta1"}:                {group: 16800, version: 9},
	{Group: "apiextensions.k8s.io", Version: "v1beta1"}:          {group: 16700, version: 9},
	{Group: "admissionregistration.k8s.io", Version: "v1alpha1"}: {group: 16700, version: 9},
}

func createAggregatorConfig(kubeAPIServerConfig genericapiserver.Config, commandOptions *options.ServerRunOptions, proxyTransport *http.Transport) (*aggregatorapiserver.Config, error) {
	// make a shallow copy to let us twiddle a few things
	// most of the config actually remains the same.  We only need to mess with a couple items related to the particulars of the aggregator
	genericConfig := kubeAPIServerConfig

	// the aggregator doesn't wire these up.  It just delegates them to the kubeapiserver
	//genericConfig.EnableSwaggerUI = false
	//genericConfig.SwaggerConfig = nil

	// copy the etcd options so we don't mutate originals.
	//etcdOptions := *commandOptions.Etcd
	////etcdOptions.StorageConfig.Codec = aggregatorapiserver.Codecs.LegacyCodec(v1beta1.SchemeGroupVersion)
	////etcdOptions.StorageConfig.Copier = aggregatorapiserver.Scheme
	////genericConfig.RESTOptionsGetter = &genericoptions.SimpleRestOptionsFactory{Options: etcdOptions}
	//
	//var err error
	var certBytes, keyBytes []byte
	//if len(commandOptions.ProxyClientCertFile) > 0 && len(commandOptions.ProxyClientKeyFile) > 0 {
	//	certBytes, err = ioutil.ReadFile(commandOptions.ProxyClientCertFile)
	//	if err != nil {
	//		return nil, err
	//	}
	//	keyBytes, err = ioutil.ReadFile(commandOptions.ProxyClientKeyFile)
	//	if err != nil {
	//		return nil, err
	//	}
	//}

	aggregatorConfig := &aggregatorapiserver.Config{
		GenericConfig:     &genericConfig,
		//CoreKubeInformers: externalInformers,
		ProxyClientCert:   certBytes,
		ProxyClientKey:    keyBytes,
		//ServiceResolver:   serviceResolver,
		ProxyTransport:    proxyTransport,
	}

	return aggregatorConfig, nil
}


func createAggregatorServer(aggregatorConfig *aggregatorapiserver.Config, delegateAPIServer genericapiserver.DelegationTarget) (*aggregatorapiserver.APIAggregator, error) {
	aggregatorServer, err := aggregatorConfig.Complete().NewWithDelegate(delegateAPIServer)
	if err != nil {
		return nil, err
	}

	// create controllers for auto-registration
	//apiRegistrationClient, err := apiregistrationclient.NewForConfig(aggregatorConfig.GenericConfig.LoopbackClientConfig)
	//if err != nil {
	//	return nil, err
	//}
	autoRegistrationController := autoregister.NewAutoRegisterController()
	//apiServices := apiServicesToRegister(delegateAPIServer, autoRegistrationController)
	//crdRegistrationController := crdregistration.NewAutoRegistrationController()

	aggregatorServer.GenericAPIServer.AddPostStartHook("kube-apiserver-autoregistration", func(context genericapiserver.PostStartHookContext) error {
		go autoRegistrationController.Run(5, context.StopCh)
		//go crdRegistrationController.Run(5, context.StopCh)
		return nil
	})
	//aggregatorServer.GenericAPIServer.AddHealthzChecks(healthz.NamedCheck("autoregister-completion", func(r *http.Request) error {
	//	items, err := aggregatorServer.APIRegistrationInformers.Apiregistration().InternalVersion().APIServices().Lister().List(labels.Everything())
	//	if err != nil {
	//		return err
	//	}
	//
	//	missing := []apiregistration.APIService{}
	//	for _, apiService := range apiServices {
	//		found := false
	//		for _, item := range items {
	//			if item.Name != apiService.Name {
	//				continue
	//			}
	//			if apiregistration.IsAPIServiceConditionTrue(item, apiregistration.Available) {
	//				found = true
	//				break
	//			}
	//		}
	//
	//		if !found {
	//			missing = append(missing, *apiService)
	//		}
	//	}
	//
	//	if len(missing) > 0 {
	//		return fmt.Errorf("missing APIService: %v", missing)
	//	}
	//	return nil
	//}))

	return aggregatorServer, nil
}

func apiServicesToRegister(delegateAPIServer genericapiserver.DelegationTarget, registration autoregister.AutoAPIServiceRegistration) []*apiregistration.APIService {
	apiServices := []*apiregistration.APIService{}

	for _, curr := range delegateAPIServer.ListedPaths() {
		if curr == "/api/v1" {
			apiService := makeAPIService(schema.GroupVersion{Group: "", Version: "v1"})
			registration.AddAPIServiceToSync(apiService)
			apiServices = append(apiServices, apiService)
			continue
		}

		if !strings.HasPrefix(curr, "/apis/") {
			continue
		}
		// this comes back in a list that looks like /apis/rbac.authorization.k8s.io/v1alpha1
		tokens := strings.Split(curr, "/")
		if len(tokens) != 4 {
			continue
		}

		apiService := makeAPIService(schema.GroupVersion{Group: tokens[2], Version: tokens[3]})
		if apiService == nil {
			continue
		}
		registration.AddAPIServiceToSync(apiService)
		apiServices = append(apiServices, apiService)
	}

	return apiServices
}


func makeAPIService(gv schema.GroupVersion) *apiregistration.APIService {
	apiServicePriority, ok := apiVersionPriorities[gv]
	if !ok {
		// if we aren't found, then we shouldn't register ourselves because it could result in a CRD group version
		// being permanently stuck in the APIServices list.
		glog.Infof("Skipping APIService creation for %v", gv)
		return nil
	}
	return &apiregistration.APIService{
		ObjectMeta: metav1.ObjectMeta{Name: gv.Version + "." + gv.Group},
		Spec: apiregistration.APIServiceSpec{
			Group:                gv.Group,
			Version:              gv.Version,
			GroupPriorityMinimum: apiServicePriority.group,
			VersionPriority:      apiServicePriority.version,
		},
	}
}
