package apiserver

import (
	"github.com/HuZhou/apiserver/pkg/storage/storagebackend"
	"fmt"
	"bytes"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type UnstructuredCopier struct{}

func (UnstructuredCopier) Copy(obj runtime.Object) (runtime.Object, error) {
	if _, ok := obj.(runtime.Unstructured); !ok {
		// Callers should not use this UnstructuredCopier for things other than Unstructured.
		// If they do, the copy they get back will become Unstructured, which can lead to
		// difficult-to-debug errors downstream. To make such errors more obvious,
		// we explicitly reject anything that isn't Unstructured.
		return nil, fmt.Errorf("UnstructuredCopier can't copy type %T", obj)
	}

	// serialize and deserialize to ensure a clean copy
	buf := &bytes.Buffer{}
	err := unstructured.UnstructuredJSONScheme.Encode(obj, buf)
	if err != nil {
		return nil, err
	}
	out := &unstructured.Unstructured{}
	result, _, err := unstructured.UnstructuredJSONScheme.Decode(buf.Bytes(), nil, out)
	return result, err
}

type CRDRESTOptionsGetter struct {
	StorageConfig           storagebackend.Config
	StoragePrefix           string
	EnableWatchCache        bool
	DefaultWatchCacheSize   int
	EnableGarbageCollection bool
	DeleteCollectionWorkers int
}