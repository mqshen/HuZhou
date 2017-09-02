package storagebackend

import (
	"k8s.io/apimachinery/pkg/runtime"
	"github.com/mqshen/HuZhou/staging/src/github.com/apiserver/pkg/storage/value"
)

type Config struct {
	// Type defines the type of storage backend, e.g. "etcd2", etcd3". Default ("") is "etcd3".
	Type string
	// Prefix is the prefix to all keys passed to storage.Interface methods.
	Prefix string
	// ServerList is the list of storage servers to connect with.
	ServerList []string
	// TLS credentials
	KeyFile  string
	CertFile string
	CAFile   string
	// Quorum indicates that whether read operations should be quorum-level consistent.
	Quorum bool
	// DeserializationCacheSize is the size of cache of deserialized objects.
	// Currently this is only supported in etcd2.
	// We will drop the cache once using protobuf.
	DeserializationCacheSize int

	Codec  runtime.Codec
	Copier runtime.ObjectCopier
	// Transformer allows the value to be transformed prior to persisting into etcd.
	Transformer value.Transformer
}

func NewDefaultConfig(prefix string, copier runtime.ObjectCopier, codec runtime.Codec) *Config {
	return &Config{
		Prefix: prefix,
		// Default cache size to 0 - if unset, its size will be set based on target
		// memory usage.
		DeserializationCacheSize: 0,
		Copier: copier,
		Codec:  codec,
	}
}