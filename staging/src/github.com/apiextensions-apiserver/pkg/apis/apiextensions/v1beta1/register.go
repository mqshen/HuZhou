package v1beta1

import "k8s.io/apimachinery/pkg/runtime/schema"

const GroupName = "apiextensions.k8s.io"

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1beta1"}