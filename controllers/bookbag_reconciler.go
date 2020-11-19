package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	"github.com/mcouliba/workshop-operator/common/bookbag"
	"github.com/mcouliba/workshop-operator/common/kubernetes"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/prometheus/common/log"

	"github.com/mcouliba/workshop-operator/common/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciling Bookbag
func (r *WorkshopReconciler) reconcileBookbag(workshop *workshopv1.Workshop, users int,
	appsHostnameSuffix string, openshiftConsoleURL string) (reconcile.Result, error) {
	enabled := workshop.Spec.Infrastructure.Guide.Bookbag.Enabled

	guidesNamespace := "workshop-guides"

	id := 1
	for {
		if id <= users && enabled {
			// Bookback
			if result, err := r.addUpdateBookbag(workshop, strconv.Itoa(id), guidesNamespace,
				appsHostnameSuffix, openshiftConsoleURL); util.IsRequeued(result, err) {
				return result, err
			}
		} else {

			bookbagName := fmt.Sprintf("bookbag-%d", id)

			depFound := &appsv1.Deployment{}
			depErr := r.Get(context.TODO(), types.NamespacedName{Name: bookbagName, Namespace: guidesNamespace}, depFound)

			if depErr != nil && errors.IsNotFound(depErr) {
				break
			}

			if result, err := r.deleteBookbag(workshop, strconv.Itoa(id), guidesNamespace); util.IsRequeued(result, err) {
				return result, err
			}
		}

		id++
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) addUpdateBookbag(workshop *workshopv1.Workshop, userID string,
	guidesNamespace string, appsHostnameSuffix string, openshiftConsoleURL string) (reconcile.Result, error) {

	namespace := kubernetes.NewNamespace(workshop, r.Scheme, guidesNamespace)
	if err := r.Create(context.TODO(), namespace); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Project", namespace.Name)
	}

	bookbagName := fmt.Sprintf("user%s-bookbag", userID)
	labels := map[string]string{
		"app":                       bookbagName,
		"app.kubernetes.io/part-of": "bookbag",
	}

	// Create ConfigMap
	data := map[string]string{
		"gateway.sh":  "",
		"terminal.sh": "",
		"workshop.sh": "",
	}

	envConfigMap := kubernetes.NewConfigMap(workshop, r.Scheme, bookbagName+"-env", namespace.Name, labels, data)
	if err := r.Create(context.TODO(), envConfigMap); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s ConfigMap", envConfigMap.Name)
	}

	varConfigMap := kubernetes.NewConfigMap(workshop, r.Scheme, bookbagName+"-vars", namespace.Name, labels, nil)
	if err := r.Create(context.TODO(), varConfigMap); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s ConfigMap", varConfigMap.Name)
	}

	// Create Service Account
	serviceAccount := kubernetes.NewServiceAccount(workshop, r.Scheme, bookbagName, namespace.Name, labels)
	if err := r.Create(context.TODO(), serviceAccount); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Service Account", serviceAccount.Name)
	}

	// Create Role Binding
	roleBinding := kubernetes.NewRoleBindingSA(workshop, r.Scheme, bookbagName, namespace.Name, labels,
		serviceAccount.Name, "adim", "Role")
	if err := r.Create(context.TODO(), roleBinding); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Role Binding", roleBinding.Name)
	}

	// Deploy/Update Bookbag
	dep := bookbag.NewDeployment(workshop, r.Scheme, bookbagName, namespace.Name, labels, userID, appsHostnameSuffix, openshiftConsoleURL)
	if err := r.Create(context.TODO(), dep); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Deployment", dep.Name)
	} else if errors.IsAlreadyExists(err) {
		deploymentFound := &appsv1.Deployment{}
		if err := r.Get(context.TODO(), types.NamespacedName{Name: dep.Name, Namespace: namespace.Name}, deploymentFound); err != nil {
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
	service := kubernetes.NewService(workshop, r.Scheme, bookbagName, namespace.Name, labels, []string{"http"}, []int32{10080})
	if err := r.Create(context.TODO(), service); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Service", service.Name)
	}

	// Create Route
	route := kubernetes.NewRoute(workshop, r.Scheme, bookbagName, namespace.Name, labels, bookbagName, 10080)
	if err := r.Create(context.TODO(), route); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Route", route.Name)
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) deleteBookbag(workshop *workshopv1.Workshop, userID string, guidesNamespace string) (reconcile.Result, error) {

	bookbagName := fmt.Sprintf("user%s-bookbag", userID)

	routeFound := &routev1.Route{}
	routeErr := r.Get(context.TODO(), types.NamespacedName{Name: bookbagName, Namespace: guidesNamespace}, routeFound)
	if routeErr == nil {
		// Delete Route
		if err := r.Delete(context.TODO(), routeFound); err != nil {
			return reconcile.Result{}, err
		}
		log.Infof("Deleted %s Route", routeFound.Name)
	}

	serviceFound := &corev1.Service{}
	serviceErr := r.Get(context.TODO(), types.NamespacedName{Name: bookbagName, Namespace: guidesNamespace}, serviceFound)
	if serviceErr == nil {
		// Delete Service
		if err := r.Delete(context.TODO(), serviceFound); err != nil {
			return reconcile.Result{}, err
		}
		log.Infof("Deleted %s Service", serviceFound.Name)
	}

	depFound := &appsv1.Deployment{}
	depErr := r.Get(context.TODO(), types.NamespacedName{Name: bookbagName, Namespace: guidesNamespace}, depFound)
	if depErr == nil {
		// Undeploy Guide
		if err := r.Delete(context.TODO(), depFound); err != nil {
			return reconcile.Result{}, err
		}
		log.Infof("Deleted %s Deployment", depFound.Name)
	}

	serviceAccountFound := &corev1.ServiceAccount{}
	serviceAccountErr := r.Get(context.TODO(), types.NamespacedName{Name: bookbagName, Namespace: guidesNamespace}, serviceAccountFound)
	if serviceAccountErr == nil {
		// Delete Service
		if err := r.Delete(context.TODO(), serviceAccountFound); err != nil {
			return reconcile.Result{}, err
		}
		log.Infof("Deleted %s Service Account", serviceAccountFound.Name)
	}

	//Success
	return reconcile.Result{}, nil
}
