package argocd

import (
	argocdoperator "github.com/argoproj-labs/argocd-operator/pkg/apis/argoproj/v1alpha1"
	argocd "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewArgoCDCustomResource create a ArgoCD Custom Resource
func NewArgoCDCustomResource(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string, labels map[string]string, argocdPolicy string) *argocdoperator.ArgoCD {

	scopes := "[preferred_username]"
	defaultPolicy := ""

	cr := &argocdoperator.ArgoCD{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: argocdoperator.ArgoCDSpec{
			ApplicationInstanceLabelKey: "argocd.argoproj.io/instance",
			// Dex: argocd.ArgoCDDexSpec{
			// 	OpenShiftOAuth: true,
			// },
			Server: argocdoperator.ArgoCDServerSpec{
				Insecure: true,
				Route: argocdoperator.ArgoCDRouteSpec{
					Enabled: true,
				},
			},
			RBAC: argocdoperator.ArgoCDRBACSpec{
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

// NewAppProjectCustomResource create a AppProject Custom Resource
func NewAppProjectCustomResource(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string, labels map[string]string, argocdPolicy string) *argocd.AppProject {

	cr := &argocd.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: argocd.AppProjectSpec{
			Destinations: []argocd.ApplicationDestination{
				{
					Namespace: name,
					Server:    "https://kubernetes.default.svc",
				},
			},
			SourceRepos: []string{
				"*",
			},
		},
	}

	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, cr, scheme)
	return cr
}
