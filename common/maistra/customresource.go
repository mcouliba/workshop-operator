package maistra

import (
	maistrav1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	maistrav2 "github.com/maistra/istio-operator/pkg/apis/maistra/v2"
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewServiceMeshControlPlaneCR create a SMCP Custom Resource
func NewServiceMeshControlPlaneCR(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string) *maistrav2.ServiceMeshControlPlane {

	var sampling int32 = 10000

	smcp := &maistrav2.ServiceMeshControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: maistrav2.ControlPlaneSpec{
			Version: "v2.0",
			Tracing: &maistrav2.TracingConfig{
				Type:     maistrav2.TracerTypeJaeger,
				Sampling: &sampling,
			},
			Policy: &maistrav2.PolicyConfig{
				Type: maistrav2.PolicyTypeIstiod,
			},
			Telemetry: &maistrav2.TelemetryConfig{
				Type: maistrav2.TelemetryTypeIstiod,
			},
			Addons: &maistrav2.AddonsConfig{
				Jaeger: &maistrav2.JaegerAddonConfig{
					Install: &maistrav2.JaegerInstallConfig{
						Storage: &maistrav2.JaegerStorageConfig{
							Type: maistrav2.JaegerStorageTypeMemory,
						},
					},
				},
				Prometheus: &maistrav2.PrometheusAddonConfig{},
				Kiali:      &maistrav2.KialiAddonConfig{},
			},
		},
	}

	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, smcp, scheme)

	return smcp
}

// NewServiceMeshMemberRollCR create a SMMR Custom Resource
func NewServiceMeshMemberRollCR(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string, members []string) *maistrav1.ServiceMeshMemberRoll {
	smmr := &maistrav1.ServiceMeshMemberRoll{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: maistrav1.ServiceMeshMemberRollSpec{
			Members: members,
		},
	}

	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, smmr, scheme)

	return smmr
}
