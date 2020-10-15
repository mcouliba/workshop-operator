package controllers

import (
	"context"

	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	"github.com/mcouliba/workshop-operator/common/kubernetes"
	"github.com/prometheus/common/log"

	"github.com/mcouliba/workshop-operator/common/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciling Pipeline
func (r *WorkshopReconciler) reconcilePipeline(workshop *workshopv1.Workshop) (reconcile.Result, error) {
	enabledPipeline := workshop.Spec.Infrastructure.Pipeline.Enabled

	if enabledPipeline {
		if result, err := r.addPipeline(workshop); util.IsRequeued(result, err) {
			return result, err
		}
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) addPipeline(workshop *workshopv1.Workshop) (reconcile.Result, error) {

	name := "openshift-pipelines-operator-rh"
	channel := workshop.Spec.Infrastructure.Pipeline.OperatorHub.Channel
	clusterServiceVersion := workshop.Spec.Infrastructure.Pipeline.OperatorHub.ClusterServiceVersion

	pipelineSubscription := kubernetes.NewRedHatSubscription(workshop, r.Scheme, name, "openshift-operators",
		name, channel, clusterServiceVersion)
	if err := r.Create(context.TODO(), pipelineSubscription); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Subscription", pipelineSubscription.Name)
	}

	// Approve the installation
	if err := r.ApproveInstallPlan(clusterServiceVersion, name, "openshift-operators"); err != nil {
		log.Infof("Waiting for Subscription to create InstallPlan for %s", name)
		return reconcile.Result{}, err
	}

	//Success
	return reconcile.Result{}, nil
}
