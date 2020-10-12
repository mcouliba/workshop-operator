package controllers

import (
	"context"
	"fmt"
	"time"

	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	"github.com/mcouliba/workshop-operator/deployment/kubernetes"
	smcp "github.com/mcouliba/workshop-operator/deployment/maistra/servicemeshcontrolplane"
	smmr "github.com/mcouliba/workshop-operator/deployment/maistra/servicemeshmemberroll"
	"github.com/mcouliba/workshop-operator/util"
	"github.com/prometheus/common/log"

	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciling ServiceMesh
func (r *WorkshopReconciler) reconcileServiceMesh(workshop *workshopv1.Workshop, users int) (reconcile.Result, error) {
	enabledServiceMesh := workshop.Spec.Infrastructure.ServiceMesh.Enabled
	enabledServerless := workshop.Spec.Infrastructure.Serverless.Enabled

	if enabledServiceMesh || enabledServerless {

		if result, err := r.addElasticSearchOperator(workshop); err != nil {
			return result, err
		}

		if result, err := r.addJaegerOperator(workshop); err != nil {
			return result, err
		}

		if result, err := r.addKialiOperator(workshop); err != nil {
			return result, err
		}

		if result, err := r.addServiceMesh(workshop, users); err != nil {
			return result, err
		}

		// Installed
		if workshop.Status.ServiceMesh != util.OperatorStatus.Installed {
			workshop.Status.ServiceMesh = util.OperatorStatus.Installed
			if err := r.Status().Update(context.TODO(), workshop); err != nil {
				log.Errorf("Failed to update Workshop status: %s", err)
				return reconcile.Result{}, nil
			}
		}
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) addServiceMesh(workshop *workshopv1.Workshop, users int) (reconcile.Result, error) {

	// Service Mesh Operator
	channel := workshop.Spec.Infrastructure.ServiceMesh.ServiceMeshOperatorHub.Channel
	clusterserviceversion := workshop.Spec.Infrastructure.ServiceMesh.ServiceMeshOperatorHub.ClusterServiceVersion

	subscription := kubernetes.NewRedHatSubscription(workshop, r.Scheme, "servicemeshoperator", "openshift-operators",
		"servicemeshoperator", channel, clusterserviceversion)
	if err := r.Create(context.TODO(), subscription); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Subscription", subscription.Name)
	}

	if err := r.ApproveInstallPlan(clusterserviceversion, "servicemeshoperator", "openshift-operators"); err != nil {
		log.Infof("Waiting for Subscription to create InstallPlan for %s", subscription.Name)
		return reconcile.Result{}, err
	}

	// Deploy Service Mesh
	istioSystemNamespace := kubernetes.NewNamespace(workshop, r.Scheme, "istio-system")
	if err := r.Create(context.TODO(), istioSystemNamespace); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Namespace", istioSystemNamespace.Name)
	}

	istioMembers := []string{}
	istioUsers := []rbac.Subject{}

	if workshop.Spec.Infrastructure.ArgoCD.Enabled {
		argocdSubject := rbac.Subject{
			Kind: rbac.UserKind,
			Name: "system:serviceaccount:argocd:argocd-application-controller",
		}
		istioUsers = append(istioUsers, argocdSubject)
	}

	for id := 1; id <= users; id++ {
		username := fmt.Sprintf("user%d", id)
		stagingProjectName := fmt.Sprintf("%s%d", workshop.Spec.Infrastructure.Project.StagingName, id)
		userSubject := rbac.Subject{
			Kind: rbac.UserKind,
			Name: username,
		}

		istioMembers = append(istioMembers, stagingProjectName)
		istioUsers = append(istioUsers, userSubject)
	}

	labels := map[string]string{
		"app.kubernetes.io/part-of": "istio",
	}

	jaegerRole := kubernetes.NewRole(workshop, r.Scheme,
		"workshop-jaeger", "istio-system", labels, kubernetes.JaegerUserRules())
	if err := r.Create(context.TODO(), jaegerRole); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Role", jaegerRole.Name)
	}

	JaegerRoleBinding := kubernetes.NewRoleBindingUsers(workshop, r.Scheme,
		"workshop-jaeger", "istio-system", labels, istioUsers, jaegerRole.Name, "Role")
	if err := r.Create(context.TODO(), JaegerRoleBinding); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Role Binding", JaegerRoleBinding.Name)
	}

	meshUserRoleBinding := kubernetes.NewRoleBindingUsers(workshop, r.Scheme,
		"mesh-users", "istio-system", labels, istioUsers, "mesh-user", "Role")

	if err := r.Create(context.TODO(), meshUserRoleBinding); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Role Binding", meshUserRoleBinding.Name)
	}

	serviceMeshControlPlaneCR := smcp.NewServiceMeshControlPlaneCR(workshop, r.Scheme,
		"full-install", istioSystemNamespace.Name)
	if err := r.Create(context.TODO(), serviceMeshControlPlaneCR); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 1}, err
	} else if err == nil {
		log.Infof("Created %s Custom Resource", serviceMeshControlPlaneCR.Name)
	}

	serviceMeshMemberRollCR := smmr.NewServiceMeshMemberRollCR(workshop, r.Scheme,
		"default", istioSystemNamespace.Name, istioMembers)
	if err := r.Create(context.TODO(), serviceMeshMemberRollCR); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Custom Resource", serviceMeshMemberRollCR.Name)
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) addElasticSearchOperator(workshop *workshopv1.Workshop) (reconcile.Result, error) {

	channel := workshop.Spec.Infrastructure.ServiceMesh.ElasticSearchOperatorHub.Channel
	clusterserviceversion := workshop.Spec.Infrastructure.ServiceMesh.ElasticSearchOperatorHub.ClusterServiceVersion

	redhatOperatorsNamespace := kubernetes.NewNamespace(workshop, r.Scheme, "openshift-operators-redhat")
	if err := r.Create(context.TODO(), redhatOperatorsNamespace); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Namespace", redhatOperatorsNamespace.Name)
	}

	subscription := kubernetes.NewRedHatSubscription(workshop, r.Scheme, "elasticsearch-operator", "openshift-operators-redhat",
		"elasticsearch-operator", channel, clusterserviceversion)
	if err := r.Create(context.TODO(), subscription); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Subscription", subscription.Name)
	}

	if err := r.ApproveInstallPlan(clusterserviceversion, "elasticsearch-operator", "openshift-operators-redhat"); err != nil {
		log.Infof("Waiting for Subscription to create InstallPlan for %s", subscription.Name)
		return reconcile.Result{}, err
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) addJaegerOperator(workshop *workshopv1.Workshop) (reconcile.Result, error) {

	channel := workshop.Spec.Infrastructure.ServiceMesh.JaegerOperatorHub.Channel
	clusterserviceversion := workshop.Spec.Infrastructure.ServiceMesh.JaegerOperatorHub.ClusterServiceVersion

	subscription := kubernetes.NewRedHatSubscription(workshop, r.Scheme, "jaeger-product", "openshift-operators",
		"jaeger-product", channel, clusterserviceversion)
	if err := r.Create(context.TODO(), subscription); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Subscription", subscription.Name)
	}

	if err := r.ApproveInstallPlan(clusterserviceversion, "jaeger-product", "openshift-operators"); err != nil {
		log.Infof("Waiting for Subscription to create InstallPlan for %s", subscription.Name)
		return reconcile.Result{}, err
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) addKialiOperator(workshop *workshopv1.Workshop) (reconcile.Result, error) {

	channel := workshop.Spec.Infrastructure.ServiceMesh.KialiOperatorHub.Channel
	clusterserviceversion := workshop.Spec.Infrastructure.ServiceMesh.KialiOperatorHub.ClusterServiceVersion

	subscription := kubernetes.NewRedHatSubscription(workshop, r.Scheme, "kiali-ossm", "openshift-operators",
		"kiali-ossm", channel, clusterserviceversion)
	if err := r.Create(context.TODO(), subscription); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Subscription", subscription.Name)
	}

	if err := r.ApproveInstallPlan(clusterserviceversion, "kiali-ossm", "openshift-operators"); err != nil {
		log.Infof("Waiting for Subscription to create InstallPlan for %s", subscription.Name)
		return reconcile.Result{}, err
	}

	//Success
	return reconcile.Result{}, nil
}
