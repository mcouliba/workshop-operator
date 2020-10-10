package kubernetes

import (
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewPersistentVolumeClaim creates a new persistent volume claim
func NewPersistentVolumeClaim(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string, labels map[string]string, pvcClaimSize string) *corev1.PersistentVolumeClaim {

	accessModes := []corev1.PersistentVolumeAccessMode{
		corev1.ReadWriteOnce,
	}
	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceName(corev1.ResourceStorage): resource.MustParse(pvcClaimSize),
		}}
	pvcSpec := corev1.PersistentVolumeClaimSpec{
		AccessModes: accessModes,
		Resources:   resources,
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: pvcSpec,
	}

	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, pvc, scheme)

	return pvc

}
