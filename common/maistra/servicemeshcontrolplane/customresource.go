package maistra

import (
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewServiceMeshControlPlaneCR create a SMCP Custom Resource
func NewServiceMeshControlPlaneCR(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string) *ServiceMeshControlPlane {
	smcp := &ServiceMeshControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: ServiceMeshControlPlaneSpec{
			Istio: IstioSpec{
				Global: GlobalSpec{
					Proxy: ProxySpec{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
				},
				Gateways: GatewaysSpec{
					IstioEgressgateway: IstioEgressgatewaySpec{
						AutoscaleEnabled: false,
					},
					IstioIngressgateway: IstioIngressgatewaySpec{
						AutoscaleEnabled: false,
						IorEnabled:       true,
					},
				},
				Mixer: MixerSpec{
					Policy: PolicySpec{
						AutoscaleEnabled: false,
					},
					Telemetry: TelemetrySpec{
						AutoscaleEnabled: false,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("1G"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("4G"),
							},
						},
					},
				},
				Pilot: PilotSpec{
					AutoscaleEnabled: false,
					TraceSampling:    100.0,
				},
				Kiali: KialiSpec{
					Enabled: true,
				},
				Grafana: GrafanaSpec{
					Enabled: true,
				},
				Tracing: TracingSpec{
					Enabled: true,
					Jaeger: JaegerSpec{
						Template: "all-in-one",
					},
				},
			},
		},
	}

	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, smcp, scheme)

	return smcp
}
