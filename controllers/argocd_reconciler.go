package controllers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	"github.com/mcouliba/workshop-operator/deployment/argocd"
	"github.com/mcouliba/workshop-operator/deployment/kubernetes"
	"github.com/mcouliba/workshop-operator/util"
	"github.com/prometheus/common/log"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciling ArgoCD
func (r *WorkshopReconciler) reconcileArgoCD(workshop *workshopv1.Workshop, users int,
	appsHostnameSuffix string, openshiftConsoleURL string) (reconcile.Result, error) {
	enabledArgoCD := workshop.Spec.Infrastructure.ArgoCD.Enabled

	if enabledArgoCD {

		if result, err := r.addArgoCD(workshop, users, appsHostnameSuffix, openshiftConsoleURL); err != nil {
			return result, err
		}
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) addArgoCD(workshop *workshopv1.Workshop, users int,
	appsHostnameSuffix string, openshiftConsoleURL string) (reconcile.Result, error) {

	channel := workshop.Spec.Infrastructure.ArgoCD.OperatorHub.Channel
	clusterServiceVersion := workshop.Spec.Infrastructure.ArgoCD.OperatorHub.ClusterServiceVersion

	namespace := kubernetes.NewNamespace(workshop, r.Scheme, "argocd")
	if err := r.Create(context.TODO(), namespace); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Project", namespace.Name)
	}

	operatorGroup := kubernetes.NewOperatorGroup(workshop, r.Scheme, "argocd-operator", namespace.Name)
	if err := r.Create(context.TODO(), operatorGroup); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s OperatorGroup", operatorGroup.Name)
	}

	subscription := kubernetes.NewCommunitySubscription(workshop, r.Scheme, "argocd-operator", namespace.Name,
		"argocd-operator", channel, clusterServiceVersion)
	if err := r.Create(context.TODO(), subscription); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Subscription", subscription.Name)
	}

	// Approve the installation
	if err := r.ApproveInstallPlan(clusterServiceVersion, "argocd-operator", namespace.Name); err != nil {
		log.Infof("Waiting for Subscription to create InstallPlan for %s", "argocd-operator")
		return reconcile.Result{Requeue: true}, nil
	}

	// Wait for ArgoCD Operator to be running
	if !kubernetes.GetK8Client().GetDeploymentStatus("argocd-operator", namespace.Name) {
		return reconcile.Result{Requeue: true}, nil
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
	}

	labels["app.kubernetes.io/name"] = "argocd-cm"
	configmap := kubernetes.NewConfigMap(workshop, r.Scheme, "argocd-cm", namespace.Name, labels, configMapData)
	if err := r.Create(context.TODO(), configmap); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s ConfigMap", configmap.Name)
	}

	labels["app.kubernetes.io/name"] = "argocd-cr"
	customResource := argocd.NewCustomResource(workshop, r.Scheme, "argocd", namespace.Name, labels, argocdPolicy)
	if err := r.Create(context.TODO(), customResource); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		log.Infof("Created %s Custom Resource", customResource.Name)
	}

	// Wait for ArgoCD Dex Server to be running
	if !kubernetes.GetK8Client().GetDeploymentStatus("argocd-dex-server", namespace.Name) {
		return reconcile.Result{Requeue: true}, nil
	}

	// Wait for ArgoCD Server to be running
	if !kubernetes.GetK8Client().GetDeploymentStatus("argocd-server", namespace.Name) {
		return reconcile.Result{Requeue: true}, nil
	}

	time.Sleep(time.Duration(10) * time.Second)
	adminToken, result, err := getAdminToken(workshop, namespace.Name, appsHostnameSuffix)
	if err != nil {
		return result, err
	}

	if result, err := createRepository(workshop, adminToken, appsHostnameSuffix); err != nil {
		return result, err
	}

	for id := 1; id <= users; id++ {
		stagingProject := fmt.Sprintf("%s%d", workshop.Spec.Infrastructure.Project.StagingName, id)

		if result, err := createApplication(workshop, stagingProject, adminToken, appsHostnameSuffix); err != nil {
			return result, err
		}
	}

	//Success
	return reconcile.Result{}, nil
}

func getAdminToken(workshop *workshopv1.Workshop, namespace string, appsHostnameSuffix string) (string, reconcile.Result, error) {
	var (
		err          error
		httpResponse *http.Response
		httpRequest  *http.Request
		sessionURL   = "https://argocd-server-argocd." + appsHostnameSuffix + "/api/v1/session"

		adminToken util.ArgoToken
		client     = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			// Do not follow Redirect
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	)

	serverPodname, err := kubernetes.GetK8Client().GetDeploymentPod("argocd-server", namespace, "app.kubernetes.io/name")
	if err == nil {
		body := "{\"username\": \"admin\", \"password\":\"" + serverPodname + "\"}"

		// GET TOKEN
		httpRequest, err = http.NewRequest("POST", sessionURL, strings.NewReader(body))

		httpResponse, err = client.Do(httpRequest)
		if err != nil {
			log.Errorf("Error when getting Argo CD token: %v", err)
			return "", reconcile.Result{}, err
		}
		defer httpResponse.Body.Close()

		if httpResponse.StatusCode == http.StatusOK {
			if err := json.NewDecoder(httpResponse.Body).Decode(&adminToken); err != nil {
				log.Errorf("Error when parsing Argo CD Token: %v", err)
				return "", reconcile.Result{}, err
			}
		} else {
			log.Errorf("Error when getting Argo CD token (%d)", httpResponse.StatusCode)
			return "", reconcile.Result{}, err
		}
	} else {
		log.Errorf("Error when getting Argo CD Server Pod Name: %v", err)
		return "", reconcile.Result{}, err
	}
	return adminToken.Token, reconcile.Result{}, nil
}

func createRepository(workshop *workshopv1.Workshop, token string,
	appsHostnameSuffix string) (reconcile.Result, error) {

	var (
		err          error
		httpResponse *http.Response
		httpRequest  *http.Request
		repoURL      = "https://argocd-server-argocd." + appsHostnameSuffix + "/api/v1/repositories"
		body         = "{\"repo\": \"" + workshop.Spec.Source.GitURL + "\"}"
		client       = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			// Do not follow Redirect
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	)

	httpRequest, err = http.NewRequest("POST", repoURL, strings.NewReader(body))
	httpRequest.Header.Set("Authorization", "Bearer "+token)
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")

	httpResponse, err = client.Do(httpRequest)
	if err != nil {
		log.Errorf("Error when creating a Argo CD Repository %s: %v", workshop.Spec.Source.GitURL, err)
		return reconcile.Result{}, err
	}
	defer httpResponse.Body.Close()

	//Success
	return reconcile.Result{}, nil
}

func createApplication(workshop *workshopv1.Workshop, stagingProject string, token string,
	appsHostnameSuffix string) (reconcile.Result, error) {

	var (
		err          error
		httpResponse *http.Response
		httpRequest  *http.Request
		repoURL      = "https://argocd-server-argocd." + appsHostnameSuffix + "/api/v1/applications"
		body         = `{
"metadata": {
		"name": "` + stagingProject + `"
	},
	"spec": {
		"source": {
			"repoURL": "` + workshop.Spec.Source.GitURL + `",
			"path": "gitops",
			"targetRevision": "` + workshop.Spec.Source.GitBranch + `"
		},
		"destination": {
			"server": "https://kubernetes.default.svc",
			"namespace": "` + stagingProject + `"
		},
		"project": "default"
	}
}`
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			// Do not follow Redirect
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	)

	httpRequest, err = http.NewRequest("POST", repoURL, strings.NewReader(body))
	httpRequest.Header.Set("Authorization", "Bearer "+token)
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")

	httpResponse, err = client.Do(httpRequest)
	if err != nil {
		log.Errorf("Error when creating a Argo CD Application for %s: %v", stagingProject, err)
		return reconcile.Result{}, err
	}
	defer httpResponse.Body.Close()

	//Success
	return reconcile.Result{}, nil
}
