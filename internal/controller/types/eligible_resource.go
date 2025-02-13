package types

import corev1 "k8s.io/api/core/v1"

type EligibleResource struct {
	ResultRef           corev1.ObjectReference
	ObjectRef           corev1.ObjectReference
	GVK                 string
	OriginConfiguration string
}
