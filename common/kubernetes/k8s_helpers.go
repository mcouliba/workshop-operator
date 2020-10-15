package kubernetes

import (
	"bytes"
	"context"
	"io"

	"github.com/prometheus/common/log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	k8sclient = GetK8Client()
)

type k8s struct {
	clientset kubernetes.Interface
}

// GetK8Client returns k8s client
func GetK8Client() *k8s {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Errorf(err.Error())
	}
	client := k8s{}
	client.clientset, err = kubernetes.NewForConfig(cfg)

	if err != nil {
		log.Errorf(err.Error())
		return nil
	}
	return &client
}

// GetObject retrieves kubernetes resource
func GetObject(client client.Client, name string, namespace string, obj runtime.Object) error {
	return client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, obj)
}

// IsObjectFound returns true if the kubernetes resource is found
func IsObjectFound(client client.Client, name string, namespace string, obj runtime.Object) bool {
	if err := GetObject(client, namespace, name, obj); err != nil {
		return false
	}
	return true
}

// GetDeploymentStatus listens to deployment events and checks replicas once MODIFIED event is received
func (cl *k8s) GetDeploymentStatus(name string, namespace string) (scaled bool) {
	api := cl.clientset.AppsV1()
	var timeout int64 = 420
	listOptions := metav1.ListOptions{
		FieldSelector:  fields.OneTermEqualSelector("metadata.name", name).String(),
		TimeoutSeconds: &timeout,
	}
	watcher, err := api.Deployments(namespace).Watch(context.TODO(), listOptions)
	if err != nil {
		log.Error(err, "An error occurred")
	}
	ch := watcher.ResultChan()
	log.Infof("Waiting for deployment %s. Default timeout: %v seconds", name, timeout)
	for event := range ch {
		dc, ok := event.Object.(*appsv1.Deployment)
		if !ok {
			log.Error(err, "Unexpected type")
		}
		// check before watching in case the deployment is already scaled to 1
		deployment, err := cl.clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			log.Errorf("Failed to get %s deployment: %s", deployment.Name, err)
			return false
		}
		if deployment.Status.ReadyReplicas == 1 {
			log.Infof("Deployment '%s' successfully scaled to %v", deployment.Name, deployment.Status.ReadyReplicas)
			return true
		}
		switch event.Type {
		case watch.Error:
			watcher.Stop()
		case watch.Modified:
			if dc.Status.ReadyReplicas == 1 {
				log.Infof("Deployment '%s' successfully scaled to %v", deployment.Name, dc.Status.ReadyReplicas)
				watcher.Stop()
				return true

			}
		}
	}
	dc, _ := cl.clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if dc.Status.ReadyReplicas != 1 {
		log.Errorf("Failed to verify a successful %s deployment", name)
		eventList := cl.GetEvents(name, namespace).Items
		for i := range eventList {
			log.Errorf("Event message: %v", eventList[i].Message)
		}
		deploymentPod, err := cl.GetDeploymentPod(name, namespace, "app")
		if err != nil {
			return false
		}
		cl.GetPodLogs(deploymentPod, namespace)
		log.Errorf("Command to get deployment logs: kubectl logs deployment/%s -n=%s", name, namespace)
		log.Errorf("Get k8s events: kubectl get events "+
			"--field-selector "+
			"involvedObject.name=$(kubectl get pods -l=component=%s -n=%s"+
			" -o=jsonpath='{.items[0].metadata.name}') -n=%s", name, namespace, namespace)
		return false
	}
	return true
}

// GetEvents returns a list of events filtered by involvedObject
func (cl *k8s) GetEvents(deploymentName string, namespace string) (list *corev1.EventList) {
	eventListOptions := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector("involvedObject.fieldPath", "spec.containers{"+deploymentName+"}").String()}
	deploymentEvents, _ := cl.clientset.CoreV1().Events(namespace).List(context.TODO(), eventListOptions)
	return deploymentEvents
}

// GetLogs prints stderr or stdout from a selected pod. Log size is capped at 60000 bytes
func (cl *k8s) GetPodLogs(podName string, namespace string) {
	var limitBytes int64 = 60000
	req := cl.clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{LimitBytes: &limitBytes})
	readCloser, err := req.Stream(context.TODO())
	if err != nil {
		log.Errorf("Pod error log: %v", err)
	} else {
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, readCloser)
		log.Infof("Pod log: %v", buf.String())
	}
}

//GetDeploymentPod queries all pods is a selected namespace by LabelSelector
func (cl *k8s) GetDeploymentPod(name string, namespace string, label string) (podName string, err error) {
	api := cl.clientset.CoreV1()
	listOptions := metav1.ListOptions{
		LabelSelector: label + "=" + name,
	}

	podList, _ := api.Pods(namespace).List(context.TODO(), listOptions)
	podListItems := podList.Items
	if len(podListItems) == 0 {
		log.Errorf("Failed to find pod to exec into. List of pods: %v", podListItems)
		return "", err
	}
	// expecting only one pod to be there so, taking the first one
	// todo maybe add a unique label to deployments?
	podName = podListItems[0].Name
	return podName, nil
}
