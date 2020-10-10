package vault

import (
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewAgentInjectorDeployment creates a Deployment
func NewAgentInjectorDeployment(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string, labels map[string]string) *appsv1.Deployment {

	image := workshop.Spec.Infrastructure.Vault.AgentInjectorImage.Name + ":" + workshop.Spec.Infrastructure.Vault.AgentInjectorImage.Tag
	vaultImage := workshop.Spec.Infrastructure.Vault.Image.Name + ":" + workshop.Spec.Infrastructure.Vault.Image.Tag

	replicas := int32(1)

	runAsNonRoot := true
	runAsGroup := int64(1000)
	runAsUser := int64(100)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: name,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &runAsNonRoot,
						RunAsGroup:   &runAsGroup,
						RunAsUser:    &runAsUser,
					},
					Containers: []corev1.Container{
						{
							Name:            "sidecar-injector",
							Image:           image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env: []corev1.EnvVar{
								{
									Name:  "AGENT_INJECT_LISTEN",
									Value: ":8080",
								},
								{
									Name:  "AGENT_INJECT_LOG_LEVEL",
									Value: "info",
								},
								{
									Name:  "AGENT_INJECT_VAULT_ADDR",
									Value: "http://vault." + namespace + ".svc:8200",
								},
								{
									Name:  "AGENT_INJECT_VAULT_AUTH_PATH",
									Value: "auth/kubernetes",
								},
								{
									Name:  "AGENT_INJECT_VAULT_IMAGE",
									Value: vaultImage,
								},
								{
									Name:  "AGENT_INJECT_TLS_AUTO",
									Value: name + "-cfg",
								},
								{
									Name:  "AGENT_INJECT_TLS_AUTO_HOSTS",
									Value: "vault-agent-injector,vault-agent-injector." + namespace + ",vault-agent-injector." + namespace + "svc",
								},
								{
									Name:  "AGENT_INJECT_LOG_FORMAT",
									Value: "standard",
								},
								{
									Name:  "AGENT_INJECT_REVOKE_ON_SHUTDOWN",
									Value: "false",
								},
							},
							Args: []string{
								"agent-inject",
								"2>&1",
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health/ready",
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: int32(8080),
										},
										Scheme: corev1.URISchemeHTTPS,
									},
								},
								InitialDelaySeconds: 1,
								FailureThreshold:    2,
								PeriodSeconds:       2,
								SuccessThreshold:    1,
								TimeoutSeconds:      5,
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health/ready",
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: int32(8080),
										},
										Scheme: corev1.URISchemeHTTPS,
									},
								},
								InitialDelaySeconds: 1,
								FailureThreshold:    2,
								PeriodSeconds:       2,
								SuccessThreshold:    1,
								TimeoutSeconds:      5,
							},
						},
					},
				},
			},
		},
	}

	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, dep, scheme)

	return dep
}
