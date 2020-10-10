package controllers

import (
	"context"

	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	"github.com/mcouliba/workshop-operator/deployment/kubernetes"
	"github.com/mcouliba/workshop-operator/deployment/vault"
	"github.com/mcouliba/workshop-operator/util"
	securityv1 "github.com/openshift/api/security/v1"
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciling Vault
func (r *WorkshopReconciler) reconcileVault(workshop *workshopv1.Workshop, users int) (reconcile.Result, error) {
	enabled := workshop.Spec.Infrastructure.Vault.Enabled

	if enabled {
		if result, err := r.addVaultServer(workshop, users); err != nil {
			return result, err
		}

		if result, err := r.addVaultAgentInjector(workshop, users); err != nil {
			return result, err
		}
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) addVaultServer(workshop *workshopv1.Workshop, users int) (reconcile.Result, error) {
	labels := map[string]string{
		"app":                       "vault",
		"app.kubernetes.io/name":    "vault",
		"app.kubernetes.io/part-of": "vault",
		"component":                 "server",
	}

	namespace := kubernetes.NewNamespace(workshop, r.Scheme, "vault")
	if err := r.Create(context.TODO(), namespace); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Project", namespace.Name)
	}

	extraconfigFromValues := map[string]string{
		"extraconfig-from-values.hcl": `disable_mlock = true
ui = true

listener "tcp" {
	tls_disable = 1
	address = "[::]:8200"
	cluster_address = "[::]:8201"
}
storage "file" {
	path = "/vault/data"
}
`,
	}

	configMap := kubernetes.NewConfigMap(workshop, r.Scheme, "vault-config", namespace.Name, labels, extraconfigFromValues)
	if err := r.Create(context.TODO(), configMap); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s ConfigMap", configMap.Name)
	}

	// Create Service Account
	serviceAccount := kubernetes.NewServiceAccount(workshop, r.Scheme, "vault", namespace.Name, labels)
	if err := r.Create(context.TODO(), serviceAccount); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Service Account", serviceAccount.Name)
	}

	serviceAccountUser := "system:serviceaccount:" + namespace.Name + ":" + serviceAccount.Name

	privilegedSCCFound := &securityv1.SecurityContextConstraints{}
	if err := r.Get(context.TODO(), types.NamespacedName{Name: "privileged"}, privilegedSCCFound); err != nil {
		return reconcile.Result{}, err
	}
	if !util.StringInSlice(serviceAccountUser, privilegedSCCFound.Users) {
		privilegedSCCFound.Users = append(privilegedSCCFound.Users, serviceAccountUser)
		if err := r.Update(context.TODO(), privilegedSCCFound); err != nil {
			return reconcile.Result{}, err
		} else if err == nil {
			log.Infof("Updated %s SCC", privilegedSCCFound.Name)
		}
	}

	// Create ClusterRole Binding
	clusterRoleBinding := kubernetes.NewClusterRoleBindingSA(workshop, r.Scheme, "vault-server-binding", namespace.Name,
		labels, serviceAccount.Name, "system:auth-delegator", "ClusterRole")
	if err := r.Create(context.TODO(), clusterRoleBinding); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Cluster Role Binding", clusterRoleBinding.Name)
	}

	// Create Service
	internalService := kubernetes.NewService(workshop, r.Scheme, "vault-internal", namespace.Name, labels, []string{"http", "internal"}, []int32{8200, 8201})
	if err := r.Create(context.TODO(), internalService); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Service", internalService.Name)
	}

	service := kubernetes.NewService(workshop, r.Scheme, "vault", namespace.Name, labels, []string{"http", "internal"}, []int32{8200, 8201})
	if err := r.Create(context.TODO(), service); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Service", service.Name)
	}

	// Create Stateful
	stateful := vault.NewStatefulSet(workshop, r.Scheme, "vault", namespace.Name, labels)
	if err := r.Create(context.TODO(), stateful); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Stateful", stateful.Name)
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) addVaultAgentInjector(workshop *workshopv1.Workshop, users int) (reconcile.Result, error) {
	labels := map[string]string{
		"app":                       "vault",
		"app.kubernetes.io/name":    "vault-agent-injector",
		"app.kubernetes.io/part-of": "vault",
		"component":                 "webhook",
	}

	namespace := kubernetes.NewNamespace(workshop, r.Scheme, "vault")
	if err := r.Create(context.TODO(), namespace); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Project", namespace.Name)
	}

	// Create Service Account
	serviceAccount := kubernetes.NewServiceAccount(workshop, r.Scheme, "vault-agent-injector", namespace.Name, labels)
	if err := r.Create(context.TODO(), serviceAccount); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Service Account", serviceAccount.Name)
	}

	serviceAccountUser := "system:serviceaccount:" + namespace.Name + ":" + serviceAccount.Name

	privilegedSCCFound := &securityv1.SecurityContextConstraints{}
	if err := r.Get(context.TODO(), types.NamespacedName{Name: "privileged"}, privilegedSCCFound); err != nil {
		return reconcile.Result{}, err
	}

	if !util.StringInSlice(serviceAccountUser, privilegedSCCFound.Users) {
		privilegedSCCFound.Users = append(privilegedSCCFound.Users, serviceAccountUser)
		if err := r.Update(context.TODO(), privilegedSCCFound); err != nil {
			return reconcile.Result{}, err
		} else if err == nil {
			log.Infof("Updated %s SCC", privilegedSCCFound.Name)
		}
	}
	// Create Cluster Role
	clusterRole := kubernetes.NewClusterRole(workshop, r.Scheme,
		"vault-agent-injector", namespace.Name, labels, kubernetes.VaultAgentInjectorRules())
	if err := r.Create(context.TODO(), clusterRole); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Cluster Role", clusterRole.Name)
	}

	clusterRoleBinding := kubernetes.NewClusterRoleBindingSA(workshop, r.Scheme, "vault-agent-injector", namespace.Name,
		labels, "vault-agent-injector", clusterRole.Name, "ClusterRole")
	if err := r.Create(context.TODO(), clusterRoleBinding); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Cluster Role Binding", clusterRoleBinding.Name)
	}

	// Create Service
	service := kubernetes.NewServiceWithTarget(workshop, r.Scheme, "vault-agent-injector", namespace.Name, labels,
		[]string{"http"}, []int32{443}, []int32{8080})
	if err := r.Create(context.TODO(), service); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Service", service.Name)
	}

	// Create Deployment
	ocpDeployment := vault.NewAgentInjectorDeployment(workshop, r.Scheme, "vault-agent-injector", namespace.Name, labels)
	if err := r.Create(context.TODO(), ocpDeployment); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Deployment", ocpDeployment.Name)
	}

	// Create
	webhooks := vault.NewAgentInjectorWebHook(namespace.Name)
	mutatingWebhookConfiguration := kubernetes.NewMutatingWebhookConfiguration(workshop, r.Scheme,
		"vault-agent-injector-cfg", labels, webhooks)
	if err := r.Create(context.TODO(), mutatingWebhookConfiguration); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Mutating Webhook Configuration", mutatingWebhookConfiguration.Name)
	}

	//Success
	return reconcile.Result{}, nil
}
