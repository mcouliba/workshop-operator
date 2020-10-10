package maistra

import (
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewServiceMeshMemberRollCR create a SMMR Custom Resource
func NewServiceMeshMemberRollCR(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string, members []string) *ServiceMeshMemberRoll {
	smmr := &ServiceMeshMemberRoll{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: ServiceMeshMemberRollSpec{
			Members: members,
		},
	}

	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, smmr, scheme)

	return smmr
}
