package controllers

import (
	"context"
	"fmt"

	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	"github.com/mcouliba/workshop-operator/deployment/kubernetes"
	"github.com/mcouliba/workshop-operator/util"
	securityv1 "github.com/openshift/api/security/v1"
	"github.com/sirupsen/logrus"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciling IstioWorkspace
func (r *WorkshopReconciler) reconcileIstioWorkspace(workshop *workshopv1.Workshop, users int) (reconcile.Result, error) {
	enabled := workshop.Spec.Infrastructure.IstioWorkspace.Enabled

	if enabled {

		if result, err := r.addIstioWorkspace(workshop, users); err != nil {
			return result, err
		}

		// Installed
		if workshop.Status.IstioWorkspace != util.OperatorStatus.Installed {
			workshop.Status.IstioWorkspace = util.OperatorStatus.Installed
			if err := r.Status().Update(context.TODO(), workshop); err != nil {
				logrus.Errorf("Failed to update Workshop status: %s", err)
				return reconcile.Result{}, err
			}
		}
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) addIstioWorkspace(workshop *workshopv1.Workshop, users int) (reconcile.Result, error) {

	imageName := workshop.Spec.Infrastructure.IstioWorkspace.Image.Name
	imageTag := workshop.Spec.Infrastructure.IstioWorkspace.Image.Tag

	labels := map[string]string{
		"app.kubernetes.io/part-of": "istio-workspace",
	}

	customResourceDefinition := kubernetes.NewCustomResourceDefinition(workshop, r.Scheme, "sessions.maistra.io", "maistra.io", "Session", "SessionList", "sessions", "session", "v1alpha1", nil, nil)
	if err := r.Create(context.TODO(), customResourceDefinition); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		logrus.Infof("Created %s Custom Resource Definition", customResourceDefinition.Name)
	}

	serviceAccount := kubernetes.NewServiceAccount(workshop, r.Scheme, "istio-workspace", workshop.Namespace, labels)
	if err := r.Create(context.TODO(), serviceAccount); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		logrus.Infof("Created %s Service Account", serviceAccount.Name)
	}

	clusterRole := kubernetes.NewClusterRole(workshop, r.Scheme, "istio-workspace", workshop.Namespace, labels, kubernetes.IstioWorkspaceRules())
	if err := r.Create(context.TODO(), clusterRole); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		logrus.Infof("Created %s Cluster Role", clusterRole.Name)
	}

	clusterRoleBinding := kubernetes.NewClusterRoleBindingSA(workshop, r.Scheme, "istio-workspace", workshop.Namespace, labels, "istio-workspace", "istio-workspace", "ClusterRole")
	if err := r.Create(context.TODO(), clusterRoleBinding); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		logrus.Infof("Created %s Cluster Role Binding", clusterRoleBinding.Name)
	}

	operator := kubernetes.NewOperatorDeployment(workshop, r.Scheme, "istio-workspace", workshop.Namespace, labels, imageName+":"+imageTag, "istio-workspace", 8383, []string{"ike"}, []string{"serve"}, nil, nil)
	if err := r.Create(context.TODO(), operator); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		logrus.Infof("Created %s Operator", operator.Name)
	}

	for id := 1; id <= users; id++ {
		username := fmt.Sprintf("user%d", id)
		stagingProjectName := fmt.Sprintf("%s%d", workshop.Spec.Infrastructure.Project.StagingName, id)

		role := kubernetes.NewRole(workshop, r.Scheme,
			username+"-istio-workspace", stagingProjectName, labels, kubernetes.IstioWorkspaceUserRules())
		if err := r.Create(context.TODO(), role); err != nil && !errors.IsAlreadyExists(err) {
			return reconcile.Result{}, err
		} else if err == nil {
			logrus.Infof("Created %s Role", role.Name)
		}

		users := []rbac.Subject{
			{
				Kind: rbac.UserKind,
				Name: username,
			},
		}

		roleBinding := kubernetes.NewRoleBindingUsers(workshop, r.Scheme,
			username+"-istio-workspace", stagingProjectName, labels, users, username+"-istio-workspace", "Role")
		if err := r.Create(context.TODO(), roleBinding); err != nil && !errors.IsAlreadyExists(err) {
			return reconcile.Result{}, err
		} else if err == nil {
			logrus.Infof("Created %s Role Binding", roleBinding.Name)
		}

		// Create SCC
		serviceAccountUser := "system:serviceaccount:" + stagingProjectName + ":default"

		privilegedSCCFound := &securityv1.SecurityContextConstraints{}
		if err := r.Get(context.TODO(), types.NamespacedName{Name: "privileged"}, privilegedSCCFound); err != nil {
			return reconcile.Result{}, err
		}

		if !util.StringInSlice(serviceAccountUser, privilegedSCCFound.Users) {
			privilegedSCCFound.Users = append(privilegedSCCFound.Users, serviceAccountUser)
			if err := r.Update(context.TODO(), privilegedSCCFound); err != nil {
				return reconcile.Result{}, err
			} else if err == nil {
				logrus.Infof("Updated %s SCC", privilegedSCCFound.Name)
			}
		}

		anyuidSCCFound := &securityv1.SecurityContextConstraints{}
		if err := r.Get(context.TODO(), types.NamespacedName{Name: "anyuid"}, anyuidSCCFound); err != nil {
			return reconcile.Result{}, err
		}

		if !util.StringInSlice(serviceAccountUser, anyuidSCCFound.Users) {
			anyuidSCCFound.Users = append(anyuidSCCFound.Users, serviceAccountUser)
			if err := r.Update(context.TODO(), anyuidSCCFound); err != nil {
				return reconcile.Result{}, err
			} else if err == nil {
				logrus.Infof("Updated %s SCC", anyuidSCCFound.Name)
			}
		}

	}

	//Success
	return reconcile.Result{}, nil
}
