package kubernetes

import (
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewClusterRoleBindingSA creates a ClusterRoleBinding for Service Account
func NewClusterRoleBindingSA(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string, labels map[string]string, serviceAccountName string, roleName string, roleKind string) *rbac.ClusterRoleBinding {

	clusterrolebinding := &rbac.ClusterRoleBinding{
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
	ctrl.SetControllerReference(workshop, clusterrolebinding, scheme)

	return clusterrolebinding
}

// NewClusterRoleBinding creates a ClusterRoleBinding for Users
func NewClusterRoleBinding(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string, labels map[string]string, username string, roleName string, roleKind string) *rbac.ClusterRoleBinding {

	clusterrolebinding := &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Subjects: []rbac.Subject{
			{
				Kind: rbac.UserKind,
				Name: username,
			},
		},
		RoleRef: rbac.RoleRef{
			Name:     roleName,
			Kind:     roleKind,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, clusterrolebinding, scheme)

	return clusterrolebinding
}
