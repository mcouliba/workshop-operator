package controllers

import (
	"context"

	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	certmanager "github.com/mcouliba/workshop-operator/deployment/certmanager"
	"github.com/mcouliba/workshop-operator/deployment/kubernetes"
	"github.com/mcouliba/workshop-operator/util"
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciling CertManager
func (r *WorkshopReconciler) reconcileCertManager(workshop *workshopv1.Workshop, users int) (reconcile.Result, error) {
	enabledCertManager := workshop.Spec.Infrastructure.CertManager.Enabled

	if enabledCertManager {
		if result, err := r.addCertManager(workshop, users); util.IsRequeued(result, err) {
			return result, err
		}
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) addCertManager(workshop *workshopv1.Workshop, users int) (reconcile.Result, error) {

	channel := workshop.Spec.Infrastructure.CertManager.OperatorHub.Channel
	clusterServiceVersion := workshop.Spec.Infrastructure.CertManager.OperatorHub.ClusterServiceVersion

	CertManagerSubscription := kubernetes.NewCertifiedSubscription(workshop, r.Scheme, "cert-manager-operator", "openshift-operators",
		"cert-manager-operator", channel, clusterServiceVersion)
	if err := r.Create(context.TODO(), CertManagerSubscription); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Subscription", CertManagerSubscription.Name)
	}

	// Approve the installation
	if err := r.ApproveInstallPlan(clusterServiceVersion, "cert-manager-operator", "openshift-operators"); err != nil {
		log.Infof("Waiting for Subscription to create InstallPlan for %s", "CertManageroperator")
		return reconcile.Result{Requeue: true}, nil
	}

	namespace := kubernetes.NewNamespace(workshop, r.Scheme, "cert-manager")
	if err := r.Create(context.TODO(), namespace); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Namespace", namespace.Name)
	}

	labels := map[string]string{
		"app.kubernetes.io/part-of": "certmanager",
	}

	customresource := certmanager.NewCustomResource(workshop, r.Scheme, "cert-manager", namespace.Name, labels)
	if err := r.Create(context.TODO(), customresource); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Custom Resource", customresource.Name)
	}

	//Success
	return reconcile.Result{}, nil
}
