package controllers

import (
	"context"

	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	"github.com/mcouliba/workshop-operator/deployment/kubernetes"
	"github.com/mcouliba/workshop-operator/util"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciling Serverless
func (r *WorkshopReconciler) reconcileServerless(workshop *workshopv1.Workshop) (reconcile.Result, error) {
	enabledServerless := workshop.Spec.Infrastructure.Serverless.Enabled

	if enabledServerless {

		if result, err := r.addServerless(workshop); err != nil {
			return result, err
		}

		// Installed
		if workshop.Status.Serverless != util.OperatorStatus.Installed {
			workshop.Status.Serverless = util.OperatorStatus.Installed
			if err := r.Status().Update(context.TODO(), workshop); err != nil {
				logrus.Errorf("Failed to update Workshop status: %s", err)
				return reconcile.Result{}, nil
			}
		}
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) addServerless(workshop *workshopv1.Workshop) (reconcile.Result, error) {

	serverlessSubscription := kubernetes.NewRedHatSubscription(workshop, r.Scheme, "serverless-operator", "openshift-operators", "serverless-operator",
		workshop.Spec.Infrastructure.Serverless.OperatorHub.Channel,
		workshop.Spec.Infrastructure.Serverless.OperatorHub.ClusterServiceVersion)
	if err := r.Create(context.TODO(), serverlessSubscription); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		logrus.Infof("Created %s Subscription", serverlessSubscription.Name)
	}

	knativeServingNamespace := kubernetes.NewNamespace(workshop, r.Scheme, "knative-serving")
	if err := r.Create(context.TODO(), knativeServingNamespace); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		logrus.Infof("Created %s Namespace", knativeServingNamespace.Name)
	}

	// TODO
	// Add  knativeServingNamespace to ServiceMeshMember
	// Create CR

	//Success
	return reconcile.Result{}, nil
}
