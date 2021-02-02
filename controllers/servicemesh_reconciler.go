package controllers

import (
	"context"
	"fmt"
	"reflect"

	maistrav1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	"github.com/mcouliba/workshop-operator/common/kubernetes"
	"github.com/mcouliba/workshop-operator/common/maistra"
	"github.com/mcouliba/workshop-operator/common/util"
	"github.com/prometheus/common/log"

	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciling ServiceMesh
func (r *WorkshopReconciler) reconcileServiceMesh(workshop *workshopv1.Workshop, users int) (reconcile.Result, error) {
	enabledServiceMesh := workshop.Spec.Infrastructure.ServiceMesh.Enabled
	enabledServerless := workshop.Spec.Infrastructure.Serverless.Enabled

	if enabledServiceMesh || enabledServerless {

		if result, err := r.addElasticSearchOperator(workshop); util.IsRequeued(result, err) {
			return result, err
		}

		if result, err := r.addJaegerOperator(workshop); util.IsRequeued(result, err) {
			return result, err
		}

		if result, err := r.addKialiOperator(workshop); util.IsRequeued(result, err) {
			return result, err
		}

		if result, err := r.addServiceMesh(workshop, users); util.IsRequeued(result, err) {
			return result, err
		}
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) addServiceMesh(workshop *workshopv1.Workshop, users int) (reconcile.Result, error) {

	operatorNamespace := "openshift-operators"

	// Service Mesh Operator
	channel := workshop.Spec.Infrastructure.ServiceMesh.ServiceMeshOperatorHub.Channel
	clusterserviceversion := workshop.Spec.Infrastructure.ServiceMesh.ServiceMeshOperatorHub.ClusterServiceVersion

	subscription := kubernetes.NewRedHatSubscription(workshop, r.Scheme, "servicemeshoperator", operatorNamespace,
		"servicemeshoperator", channel, clusterserviceversion)
	if err := r.Create(context.TODO(), subscription); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Subscription", subscription.Name)
	}

	if err := r.ApproveInstallPlan(clusterserviceversion, "servicemeshoperator", operatorNamespace); err != nil {
		log.Infof("Waiting for Subscription to create InstallPlan for %s", subscription.Name)
		return reconcile.Result{Requeue: true}, nil
	}

	// Wait for Operator to be running
	if !kubernetes.GetK8Client().GetDeploymentStatus("istio-operator", operatorNamespace) {
		return reconcile.Result{Requeue: true}, nil
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
			Kind:     rbac.UserKind,
			Name:     "system:serviceaccount:argocd:argocd-application-controller",
			APIGroup: "rbac.authorization.k8s.io",
		}
		istioUsers = append(istioUsers, argocdSubject)
	}

	for id := 1; id <= users; id++ {
		username := fmt.Sprintf("user%d", id)
		stagingProjectName := fmt.Sprintf("%s%d", workshop.Spec.Infrastructure.Project.StagingName, id)
		userSubject := rbac.Subject{
			Kind:     rbac.UserKind,
			Name:     username,
			APIGroup: "rbac.authorization.k8s.io",
		}

		istioMembers = append(istioMembers, stagingProjectName)
		istioUsers = append(istioUsers, userSubject)
	}

	labels := map[string]string{
		"app.kubernetes.io/part-of": "istio",
	}

	// jaegerRole := kubernetes.NewRole(workshop, r.Scheme,
	// 	"workshop-jaeger", "istio-system", labels, kubernetes.JaegerUserRules())
	// if err := r.Create(context.TODO(), jaegerRole); err != nil && !errors.IsAlreadyExists(err) {
	// 	return reconcile.Result{}, err
	// } else if err == nil {
	// 	log.Infof("Created %s Role", jaegerRole.Name)
	// }

	// jaegerRoleBinding := kubernetes.NewRoleBindingUsers(workshop, r.Scheme,
	// 	"workshop-jaeger", "istio-system", labels, istioUsers, jaegerRole.Name, "Role")
	// if err := r.Create(context.TODO(), jaegerRoleBinding); err != nil && !errors.IsAlreadyExists(err) {
	// 	return reconcile.Result{}, err
	// } else if err == nil {
	// 	log.Infof("Created %s Role Binding", jaegerRoleBinding.Name)
	// } else if errors.IsAlreadyExists(err) {
	// 	found := &rbac.RoleBinding{}
	// 	if err := r.Get(context.TODO(), types.NamespacedName{Name: jaegerRoleBinding.Name, Namespace: istioSystemNamespace.Name}, found); err != nil {
	// 		return reconcile.Result{}, err
	// 	} else if err == nil {
	// 		if !reflect.DeepEqual(istioUsers, found.Subjects) {
	// 			found.Subjects = istioUsers
	// 			if err := r.Update(context.TODO(), found); err != nil {
	// 				return reconcile.Result{}, err
	// 			}
	// 			log.Infof("Updated %s Role Binding", found.Name)
	// 		}
	// 	}
	// }

	meshUserRoleBinding := kubernetes.NewRoleBindingUsers(workshop, r.Scheme,
		"mesh-users", "istio-system", labels, istioUsers, "mesh-user", "Role")

	if err := r.Create(context.TODO(), meshUserRoleBinding); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Role Binding", meshUserRoleBinding.Name)
	}

	serviceMeshControlPlaneCR := maistra.NewServiceMeshControlPlaneCR(workshop, r.Scheme, "basic", istioSystemNamespace.Name)
	if err := r.Create(context.TODO(), serviceMeshControlPlaneCR); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Service Mesh Control Plane Custom Resource", serviceMeshControlPlaneCR.Name)
	}

	serviceMeshMemberRollCR := maistra.NewServiceMeshMemberRollCR(workshop, r.Scheme,
		"default", istioSystemNamespace.Name, istioMembers)
	if err := r.Create(context.TODO(), serviceMeshMemberRollCR); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Custom Resource", serviceMeshMemberRollCR.Name)
	} else if errors.IsAlreadyExists(err) {
		serviceMeshMemberRollCRFound := &maistrav1.ServiceMeshMemberRoll{}
		if err := r.Get(context.TODO(), types.NamespacedName{Name: serviceMeshMemberRollCR.Name, Namespace: istioSystemNamespace.Name}, serviceMeshMemberRollCRFound); err != nil {
			return reconcile.Result{}, err
		} else if err == nil {
			if !reflect.DeepEqual(istioMembers, serviceMeshMemberRollCRFound.Spec.Members) {
				serviceMeshMemberRollCRFound.Spec.Members = istioMembers
				if err := r.Update(context.TODO(), serviceMeshMemberRollCRFound); err != nil {
					return reconcile.Result{}, err
				}
				log.Infof("Updated %s Service Mesh Member Roll Custom Resource", serviceMeshMemberRollCRFound.Name)
			}
		}
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
		return reconcile.Result{Requeue: true}, nil
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
		return reconcile.Result{Requeue: true}, nil
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
		return reconcile.Result{Requeue: true}, nil
	}

	//Success
	return reconcile.Result{}, nil
}
