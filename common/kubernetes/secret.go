package kubernetes

import (
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewStringDataSecret create a String Data Secret
func NewStringDataSecret(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string, labels map[string]string, stringData map[string]string) *corev1.Secret {

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		StringData: stringData,
	}

	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, secret, scheme)

	return secret
}

// NewCrtSecret create a CRT Secret
func NewCrtSecret(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string, labels map[string]string, crt []byte) *corev1.Secret {

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: map[string][]byte{
			"ca.crt": crt,
		},
	}
	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, secret, scheme)

	return secret
}
