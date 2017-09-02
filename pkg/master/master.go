package master

import (
	genericapiserver "github.com/HuZhou/apiserver/pkg/server"
)

type ClientCARegistrationHook struct {
	ClientCA []byte

	RequestHeaderUsernameHeaders     []string
	RequestHeaderGroupHeaders        []string
	RequestHeaderExtraHeaderPrefixes []string
	RequestHeaderCA                  []byte
	RequestHeaderAllowedNames        []string
}

type Config struct {
	GenericConfig *genericapiserver.Config

}

// Master contains state for a Kubernetes cluster master/api server.
type Master struct {
	GenericAPIServer *genericapiserver.GenericAPIServer

	ClientCARegistrationHook ClientCARegistrationHook
}

type completedConfig struct {
	*Config
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (c *Config) Complete() completedConfig {
	c.GenericConfig.Complete()
	return completedConfig{c}
}

func (c completedConfig) New(delegationTarget genericapiserver.DelegationTarget) (*Master, error) {
	s, err := c.Config.GenericConfig.SkipComplete().New("kube-apiserver", delegationTarget) // completion is done in Complete, no need for a second time
	if err != nil {
		return nil, err
	}
	m := &Master{
		GenericAPIServer: s,
	}
	return m, nil
}