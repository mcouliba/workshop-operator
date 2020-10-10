package kubernetes

import (
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	olmv1 "github.com/operator-framework/api/pkg/operators/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewOperatorGroup creates an Operator Group
func NewOperatorGroup(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string) *olmv1.OperatorGroup {

	operatorgroup := &olmv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: olmv1.OperatorGroupSpec{
			TargetNamespaces: []string{
				namespace,
			},
		},
	}

	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, operatorgroup, scheme)

	return operatorgroup
}
