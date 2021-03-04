package controllers

import (
	"context"
	"fmt"
	"reflect"

	argocdv1 "github.com/argoproj-labs/argocd-operator/pkg/apis/argoproj/v1alpha1"
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	"github.com/mcouliba/workshop-operator/common/argocd"
	"github.com/mcouliba/workshop-operator/common/kubernetes"
	"github.com/prometheus/common/log"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"

	"github.com/mcouliba/workshop-operator/common/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciling GitOps
func (r *WorkshopReconciler) reconcileGitOps(workshop *workshopv1.Workshop, users int,
	appsHostnameSuffix string, openshiftConsoleURL string) (reconcile.Result, error) {
	enabledGitOps := workshop.Spec.Infrastructure.GitOps.Enabled

	if enabledGitOps {
		if result, err := r.addGitOps(workshop, users, appsHostnameSuffix, openshiftConsoleURL); util.IsRequeued(result, err) {
			return result, err
		}
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) addGitOps(workshop *workshopv1.Workshop, users int,
	appsHostnameSuffix string, openshiftConsoleURL string) (reconcile.Result, error) {

	name := "openshift-gitops-operator"
	operatorNamespace := "openshift-operators"
	channel := workshop.Spec.Infrastructure.GitOps.OperatorHub.Channel
	clusterServiceVersion := workshop.Spec.Infrastructure.GitOps.OperatorHub.ClusterServiceVersion

	subscription := kubernetes.NewRedHatSubscription(workshop, r.Scheme, name, operatorNamespace,
		name, channel, clusterServiceVersion)
	if err := r.Create(context.TODO(), subscription); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Subscription", subscription.Name)
	}

	// Approve the installation
	if err := r.ApproveInstallPlan(clusterServiceVersion, name, operatorNamespace); err != nil {
		log.Infof("Waiting for Subscription to create InstallPlan for %s", name)
		return reconcile.Result{Requeue: true}, nil
	}

	// Wait for Operator to be running
	if !kubernetes.GetK8Client().GetDeploymentStatus("gitops-operator", operatorNamespace) {
		return reconcile.Result{Requeue: true}, nil
	}

	namespace := kubernetes.NewNamespace(workshop, r.Scheme, "argocd")
	if err := r.Create(context.TODO(), namespace); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Project", namespace.Name)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(workshop.Spec.User.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Errorf("Error when Bcrypt encrypt password for Argo CD: %v", err)
		return reconcile.Result{}, err
	}
	bcryptPassword := string(hashedPassword)

	argocdPolicy := ""
	secretData := map[string]string{}
	configMapData := map[string]string{}

	for id := 1; id <= users; id++ {
		username := fmt.Sprintf("user%d", id)
		userRole := fmt.Sprintf("role:%s", username)
		projectName := fmt.Sprintf("%s%d", workshop.Spec.Infrastructure.Project.StagingName, id)

		userPolicy := `p, ` + userRole + `, applications, *, default/` + projectName + `, allow
p, ` + userRole + `, clusters, get, https://kubernetes.default.svc, allow
p, ` + userRole + `, projects, get, default, allow
p, ` + userRole + `, repositories, get, ` + workshop.Spec.Source.GitURL + `, allow
p, ` + userRole + `, repositories, get, http://gitea-server.gitea.svc:3000/` + username + `/gitops-cn-project.git, allow
p, ` + userRole + `, repositories, create, http://gitea-server.gitea.svc:3000/` + username + `/gitops-cn-project.git, allow
p, ` + userRole + `, repositories, delete, http://gitea-server.gitea.svc:3000/` + username + `/gitops-cn-project.git, allow
g, ` + username + `, ` + userRole + `
`
		argocdPolicy = fmt.Sprintf("%s%s", argocdPolicy, userPolicy)

		secretData[fmt.Sprintf("accounts.%s.password", username)] = bcryptPassword

		configMapData[fmt.Sprintf("accounts.%s", username)] = "login"
	}

	labels := map[string]string{
		"app.kubernetes.io/name":    "argocd-secret",
		"app.kubernetes.io/part-of": "argocd",
	}

	secret := kubernetes.NewStringDataSecret(workshop, r.Scheme, "argocd-secret", namespace.Name, labels, secretData)
	if err := r.Create(context.TODO(), secret); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Secret", secret.Name)
		// } else if errors.IsAlreadyExists(err) {
		// 	secretFound := &corev1.Secret{}
		// 	if err := r.Get(context.TODO(), types.NamespacedName{Name: secret.Name, Namespace: namespace.Name}, secretFound); err != nil {
		// 		return reconcile.Result{}, err
		// 	} else if err == nil {
		// 		if !util.IsIntersectMap(secretData, secretFound.StringData) {
		// 			secretFound.StringData = secretData
		// 			if err := r.Update(context.TODO(), secretFound); err != nil {
		// 				return reconcile.Result{}, err
		// 			}
		// 			log.Infof("Updated %s Secret", secretFound.Name)
		// 		}
		// 	}
	}

	labels["app.kubernetes.io/name"] = "argocd-cm"
	configmap := kubernetes.NewConfigMap(workshop, r.Scheme, "argocd-cm", namespace.Name, labels, configMapData)
	if err := r.Create(context.TODO(), configmap); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s ConfigMap", configmap.Name)
	} else if errors.IsAlreadyExists(err) {
		configmapFound := &corev1.ConfigMap{}
		if err := r.Get(context.TODO(), types.NamespacedName{Name: configmap.Name, Namespace: namespace.Name}, configmapFound); err != nil {
			return reconcile.Result{}, err
		} else if err == nil {
			if !util.IsIntersectMap(configMapData, configmapFound.Data) {
				configmapFound.Data = configMapData
				if err := r.Update(context.TODO(), configmapFound); err != nil {
					return reconcile.Result{}, err
				}
				log.Infof("Updated %s ConfigMap", configmapFound.Name)
			}
		}
	}

	labels["app.kubernetes.io/name"] = "argocd-cr"
	customResource := argocd.NewCustomResource(workshop, r.Scheme, "argocd", namespace.Name, labels, argocdPolicy)
	if err := r.Create(context.TODO(), customResource); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Custom Resource", customResource.Name)
	} else if errors.IsAlreadyExists(err) {
		customResourceFound := &argocdv1.ArgoCD{}
		if err := r.Get(context.TODO(), types.NamespacedName{Name: customResource.Name, Namespace: namespace.Name}, customResourceFound); err != nil {
			return reconcile.Result{}, err
		} else if err == nil {
			if !reflect.DeepEqual(&argocdPolicy, customResourceFound.Spec.RBAC.Policy) {
				customResourceFound.Spec.RBAC.Policy = &argocdPolicy
				if err := r.Update(context.TODO(), customResourceFound); err != nil {
					return reconcile.Result{}, err
				}
				log.Infof("Updated %s Custom Resource", customResourceFound.Name)
			}
		}
	}

	// Wait for ArgoCD Dex Server to be running
	if !kubernetes.GetK8Client().GetDeploymentStatus("argocd-dex-server", namespace.Name) {
		return reconcile.Result{Requeue: true}, nil
	}

	// Wait for ArgoCD Server to be running
	if !kubernetes.GetK8Client().GetDeploymentStatus("argocd-server", namespace.Name) {
		return reconcile.Result{Requeue: true}, nil
	}

	//Success
	return reconcile.Result{}, nil
}
