package controllers

import (
	"context"

	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	"github.com/mcouliba/workshop-operator/common/kubernetes"
	"github.com/mcouliba/workshop-operator/common/util"
	"github.com/prometheus/common/log"

	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciling Serverless
func (r *WorkshopReconciler) reconcileServerless(workshop *workshopv1.Workshop) (reconcile.Result, error) {
	enabledServerless := workshop.Spec.Infrastructure.Serverless.Enabled

	if enabledServerless {

		if result, err := r.addServerless(workshop); util.IsRequeued(result, err) {
			return result, err
		}
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) addServerless(workshop *workshopv1.Workshop) (reconcile.Result, error) {

	channel := workshop.Spec.Infrastructure.Serverless.OperatorHub.Channel
	clusterServiceVersion := workshop.Spec.Infrastructure.Serverless.OperatorHub.ClusterServiceVersion

	namespace := kubernetes.NewNamespace(workshop, r.Scheme, "openshift-serverless")
	if err := r.Create(context.TODO(), namespace); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Project", namespace.Name)
	}

	subscription := kubernetes.NewRedHatSubscription(workshop, r.Scheme, "serverless-operator", namespace.Name, "serverless-operator",
		channel, clusterServiceVersion)
	if err := r.Create(context.TODO(), subscription); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Subscription", subscription.Name)
	}

	knativeServingNamespace := kubernetes.NewNamespace(workshop, r.Scheme, "knative-serving")
	if err := r.Create(context.TODO(), knativeServingNamespace); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Namespace", knativeServingNamespace.Name)
	}

	knativeEventingNamespace := kubernetes.NewNamespace(workshop, r.Scheme, "knative-eventing")
	if err := r.Create(context.TODO(), knativeEventingNamespace); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Namespace", knativeEventingNamespace.Name)
	}

	// TODO
	// Add  knativeServingNamespace to ServiceMeshMember
	// Create CR

	//Success
	return reconcile.Result{}, nil
}
