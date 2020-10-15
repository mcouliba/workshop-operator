package redis

import (
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewDeployment create a deployment
func NewDeployment(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string, labels map[string]string) *appsv1.Deployment {

	image := "image-registry.openshift-image-registry.svc:5000/openshift/redis:5"

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: name + "-data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: name,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name: name,
							Env: []corev1.EnvVar{
								{
									Name: "REDIS_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											Key: "database-password",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: name,
											},
										},
									},
								},
							},
							Image:           image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"/bin/sh",
											"-i",
											"-c",
											"test \"$(redis-cli -h 127.0.0.1 -a $REDIS_PASSWORD ping)\" == \"PONG\"",
										},
									},
								},
								InitialDelaySeconds: 5,
								FailureThreshold:    10,
								TimeoutSeconds:      1,
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: int32(6379),
										},
									},
								},
								InitialDelaySeconds: 30,
								FailureThreshold:    3,
								TimeoutSeconds:      1,
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 6379,
									Protocol:      "TCP",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      name + "-data",
									MountPath: "/var/lib/redis/data",
								},
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
