package kubernetes

import (
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewRole creates a Role
func NewRole(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string, labels map[string]string, rules []rbac.PolicyRule) *rbac.Role {

	role := &rbac.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Rules: rules,
	}

	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, role, scheme)

	return role
}
