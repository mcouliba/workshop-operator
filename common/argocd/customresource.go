package argocd

import (
	argocd "github.com/argoproj-labs/argocd-operator/pkg/apis/argoproj/v1alpha1"
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewCustomResource create a Custom Resource
func NewCustomResource(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string, labels map[string]string, argocdPolicy string) *argocd.ArgoCD {

	scopes := "[preferred_username]"
	defaultPolicy := ""

	cr := &argocd.ArgoCD{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: argocd.ArgoCDSpec{
			ApplicationInstanceLabelKey: "argocd.argoproj.io/instance",
			// Dex: argocd.ArgoCDDexSpec{
			// 	OpenShiftOAuth: true,
			// },
			Server: argocd.ArgoCDServerSpec{
				Insecure: true,
				Route: argocd.ArgoCDRouteSpec{
					Enabled: true,
				},
			},
			RBAC: argocd.ArgoCDRBACSpec{
				Policy:        &argocdPolicy,
				Scopes:        &scopes,
				DefaultPolicy: &defaultPolicy,
			},
		},
	}

	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, cr, scheme)
	return cr
}
