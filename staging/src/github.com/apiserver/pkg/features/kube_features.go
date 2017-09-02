package features

import (
	utilfeature "github.com/HuZhou/apiserver/pkg/util/feature"
)

const (
	// Every feature gate should add method here following this template:
	//
	// // owner: @username
	// // alpha: v1.4
	// MyFeature() bool

	// owner: @tallclair
	// alpha: v1.5
	//
	// StreamingProxyRedirects controls whether the apiserver should intercept (and follow)
	// redirects from the backend (Kubelet) for streaming requests (exec/attach/port-forward).
	StreamingProxyRedirects utilfeature.Feature = "StreamingProxyRedirects"

	// owner: @tallclair
	// alpha: v1.7
	//
	// AdvancedAuditing enables a much more general API auditing pipeline, which includes support for
	// pluggable output backends and an audit policy specifying how different requests should be
	// audited.
	AdvancedAuditing utilfeature.Feature = "AdvancedAuditing"

	// owner: @ilackams
	// alpha: v1.7
	//
	// Enables compression of REST responses (GET and LIST only)
	APIResponseCompression utilfeature.Feature = "APIResponseCompression"

	// owner: @smarterclayton
	// alpha: v1.7
	//
	// Allow asynchronous coordination of object creation.
	// Auto-enabled by the Initializers admission plugin.
	Initializers utilfeature.Feature = "Initializers"
)