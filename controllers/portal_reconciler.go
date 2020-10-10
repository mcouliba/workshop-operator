package controllers

import (
	"context"
	"reflect"

	"github.com/mcouliba/workshop-operator/deployment/kubernetes"
	"github.com/mcouliba/workshop-operator/deployment/redis"
	"github.com/mcouliba/workshop-operator/deployment/usernamedistribution"
	"github.com/prometheus/common/log"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
)

// reconcilePortal reconciles Portal
func (r *WorkshopReconciler) reconcilePortal(workshop *workshopv1.Workshop, users int,
	appsHostnameSuffix string, openshiftConsoleURL string) (reconcile.Result, error) {

	if result, err := r.addRedis(workshop); err != nil {
		return result, err
	}

	if result, err := r.addUpdateUsernameDistribution(workshop, users, appsHostnameSuffix, openshiftConsoleURL); err != nil {
		return result, err
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) addRedis(workshop *workshopv1.Workshop) (reconcile.Result, error) {

	serviceName := "redis"
	labels := map[string]string{
		"app":                       serviceName,
		"app.kubernetes.io/part-of": "portal",
	}

	credentials := map[string]string{
		"database-password": "redis",
	}
	secret := kubernetes.NewStringDataSecret(workshop, r.Scheme, serviceName, workshop.Namespace, labels, credentials)
	if err := r.Create(context.TODO(), secret); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Secret", secret.Name)
	}

	persistentVolumeClaim := kubernetes.NewPersistentVolumeClaim(workshop, r.Scheme, serviceName, workshop.Namespace, labels, "512Mi")
	if err := r.Create(context.TODO(), persistentVolumeClaim); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Persistent Volume Claim", persistentVolumeClaim.Name)
	}

	// Deploy/Update UsernameDistribution
	dep := redis.NewDeployment(workshop, r.Scheme, "redis", workshop.Namespace, labels)
	if err := r.Create(context.TODO(), dep); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Deployment", dep.Name)
	} else if errors.IsAlreadyExists(err) {
		deploymentFound := &appsv1.Deployment{}
		if err := r.Get(context.TODO(), types.NamespacedName{Name: dep.Name, Namespace: workshop.Namespace}, deploymentFound); err != nil {
			return reconcile.Result{}, err
		} else if err == nil {
			if !reflect.DeepEqual(dep.Spec.Template.Spec.Containers[0].Env, deploymentFound.Spec.Template.Spec.Containers[0].Env) {
				// Update Guide
				if err := r.Update(context.TODO(), dep); err != nil {
					return reconcile.Result{}, err
				}
				log.Infof("Updated %s Deployment", dep.Name)
			}
		}
	}

	// Create Service
	service := kubernetes.NewService(workshop, r.Scheme, serviceName, workshop.Namespace, labels, []string{"http"}, []int32{6379})
	if err := r.Create(context.TODO(), service); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Service", service.Name)
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) addUpdateUsernameDistribution(workshop *workshopv1.Workshop,
	users int, appsHostnameSuffix string, openshiftConsoleURL string) (reconcile.Result, error) {

	serviceName := "portal"
	redisServiceName := "redis"
	labels := map[string]string{
		"app":                       serviceName,
		"app.kubernetes.io/part-of": "portal",
	}

	// Deploy/Update UsernameDistribution
	dep := usernamedistribution.NewDeployment(workshop, r.Scheme, serviceName, labels, redisServiceName, users, appsHostnameSuffix)
	if err := r.Create(context.TODO(), dep); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Deployment", dep.Name)
	} else if errors.IsAlreadyExists(err) {
		deploymentFound := &appsv1.Deployment{}
		if err := r.Get(context.TODO(), types.NamespacedName{Name: dep.Name, Namespace: workshop.Namespace}, deploymentFound); err != nil {
			return reconcile.Result{}, err
		} else if err == nil {
			if !reflect.DeepEqual(dep.Spec.Template.Spec.Containers[0].Env, deploymentFound.Spec.Template.Spec.Containers[0].Env) ||
				!reflect.DeepEqual(dep.Spec.Template.Spec.Containers[0].Image, deploymentFound.Spec.Template.Spec.Containers[0].Image) {
				// Update Guide
				if err := r.Update(context.TODO(), dep); err != nil {
					return reconcile.Result{}, err
				}
				log.Infof("Updated %s Deployment", dep.Name)
			}
		}
	}

	// Create Service
	service := kubernetes.NewService(workshop, r.Scheme, serviceName, workshop.Namespace, labels, []string{"http"}, []int32{8080})
	if err := r.Create(context.TODO(), service); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Service", service.Name)
	}

	// Create Route
	route := kubernetes.NewRoute(workshop, r.Scheme, serviceName, workshop.Namespace, labels, serviceName, 8080)
	if err := r.Create(context.TODO(), route); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Route", route.Name)
	}

	//Success
	return reconcile.Result{}, nil
}
