package gitea

import (
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func NewCustomResource(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string, labels map[string]string) *Gitea {
	cr := &Gitea{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: GiteaSpec{
			GiteaVolumeSize:      "4Gi",
			GiteaSsl:             true,
			PostgresqlVolumeSize: "4Gi",
		},
	}

	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, cr, scheme)

	return cr
}
