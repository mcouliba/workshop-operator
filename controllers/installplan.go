package controllers

import (
	"context"
	"errors"

	"github.com/mcouliba/workshop-operator/common/kubernetes"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/prometheus/common/log"
)

// ApproveInstallPlan approves manually the install of a specific CSV
func (r *WorkshopReconciler) ApproveInstallPlan(clusterServiceVersion string, subscriptionName string, namespace string) error {

	subscription := &olmv1alpha1.Subscription{}
	if err := kubernetes.GetObject(r, subscriptionName, namespace, subscription); err != nil {
		return err
	}

	if (clusterServiceVersion == "" && subscription.Status.InstalledCSV == "") || 
		(clusterServiceVersion != "" && (subscription.Status.InstalledCSV != clusterServiceVersion || subscription.Status.CurrentCSV != clusterServiceVersion)) {
		if subscription.Status.InstallPlanRef == nil {
			return errors.New("InstallPlan Approval: Subscription is not ready yet")
		}

		installPlan := &olmv1alpha1.InstallPlan{}
		if err := kubernetes.GetObject(r, subscription.Status.InstallPlanRef.Name, namespace, installPlan); err != nil {
			return err
		}

		if !installPlan.Spec.Approved {
			installPlan.Spec.Approved = true
			if err := r.Update(context.TODO(), installPlan); err != nil {
				return err
			}
			log.Infof("%s Subscription in %s project Approved", subscriptionName, namespace)
		}
	}
	return nil
}
