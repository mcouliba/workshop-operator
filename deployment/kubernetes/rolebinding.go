package kubernetes

import (
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewRoleBindingSA creates a Role Binding for Service Account
func NewRoleBindingSA(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string, labels map[string]string,
	serviceAccountName string, roleName string, roleKind string) *rbac.RoleBinding {

	rolebinding := &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Subjects: []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      serviceAccountName,
				Namespace: namespace,
			},
		},
		RoleRef: rbac.RoleRef{
			Name:     roleName,
			Kind:     roleKind,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, rolebinding, scheme)

	return rolebinding
}

// NewRoleBindingUsers creates a Role Binding for Users
func NewRoleBindingUsers(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string, labels map[string]string,
	subject []rbac.Subject, roleName string, roleKind string) *rbac.RoleBinding {

	rolebinding := &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Subjects: subject,
		RoleRef: rbac.RoleRef{
			Name: roleName,
			Kind: roleKind,
		},
	}

	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, rolebinding, scheme)

	return rolebinding
}
