package kubernetes

import (
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
)

// NewNamespace creates a new namespace/project
func NewNamespace(workshop *workshopv1.Workshop, scheme *runtime.Scheme, name string) *corev1.Namespace {

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, namespace, scheme)

	return namespace
}
