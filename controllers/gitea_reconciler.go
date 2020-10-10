package controllers

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	"github.com/mcouliba/workshop-operator/deployment/gitea"
	"github.com/mcouliba/workshop-operator/deployment/kubernetes"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciling Gitea
func (r *WorkshopReconciler) reconcileGitea(workshop *workshopv1.Workshop, users int) (reconcile.Result, error) {
	enabledGitea := workshop.Spec.Infrastructure.Gitea.Enabled

	if enabledGitea {

		if result, err := r.addGitea(workshop, users); err != nil {
			return result, err
		}
	}

	//Success
	return reconcile.Result{}, nil
}

func (r *WorkshopReconciler) addGitea(workshop *workshopv1.Workshop, users int) (reconcile.Result, error) {

	imageName := workshop.Spec.Infrastructure.Gitea.Image.Name
	imageTag := workshop.Spec.Infrastructure.Gitea.Image.Tag

	labels := map[string]string{
		"app.kubernetes.io/part-of": "gitea",
	}

	giteaNamespace := kubernetes.NewNamespace(workshop, r.Scheme, "gitea")
	if err := r.Create(context.TODO(), giteaNamespace); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		logrus.Infof("Created %s Project", giteaNamespace.Name)
	}

	giteaCustomResourceDefinition := kubernetes.NewCustomResourceDefinition(workshop, r.Scheme, "giteas.gpte.opentlc.com", "gpte.opentlc.com", "Gitea", "GiteaList", "giteas", "gitea", "v1alpha1", nil, nil)
	if err := r.Create(context.TODO(), giteaCustomResourceDefinition); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		logrus.Infof("Created %s Custom Resource Definition", giteaCustomResourceDefinition.Name)
	}

	giteaServiceAccount := kubernetes.NewServiceAccount(workshop, r.Scheme, "gitea-operator", giteaNamespace.Name, labels)
	if err := r.Create(context.TODO(), giteaServiceAccount); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		logrus.Infof("Created %s Service Account", giteaServiceAccount.Name)
	}

	giteaClusterRole := kubernetes.NewClusterRole(workshop, r.Scheme, "gitea-operator", giteaNamespace.Name, labels, kubernetes.GiteaRules())
	if err := r.Create(context.TODO(), giteaClusterRole); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		logrus.Infof("Created %s Cluster Role", giteaClusterRole.Name)
	}

	giteaClusterRoleBinding := kubernetes.NewClusterRoleBindingSA(workshop, r.Scheme, "gitea-operator", giteaNamespace.Name, labels, "gitea-operator", "gitea-operator", "ClusterRole")
	if err := r.Create(context.TODO(), giteaClusterRoleBinding); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		logrus.Infof("Created %s Cluster Role Binding", giteaClusterRoleBinding.Name)
	}

	giteaOperator := kubernetes.NewAnsibleOperatorDeployment(workshop, r.Scheme, "gitea-operator", giteaNamespace.Name, labels, imageName+":"+imageTag, "gitea-operator")

	if err := r.Create(context.TODO(), giteaOperator); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		logrus.Infof("Created %s Operator", giteaOperator.Name)
	}

	giteaCustomResource := gitea.NewCustomResource(workshop, r.Scheme, "gitea-server", giteaNamespace.Name, labels)
	if err := r.Create(context.TODO(), giteaCustomResource); err != nil && !errors.IsAlreadyExists(err) {
		return reconcile.Result{}, err
	} else if err == nil {
		logrus.Infof("Created %s Custom Resource", giteaCustomResource.Name)
	}

	// Wait for Server to be running
	if !kubernetes.GetK8Client().GetDeploymentStatus("gitea-server", giteaNamespace.Name) {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 1}, nil
	}

	// extract app route suffix from openshift-console
	giteaRouteFound := &routev1.Route{}
	if err := r.Get(context.TODO(), types.NamespacedName{Name: "gitea-server", Namespace: giteaNamespace.Name}, giteaRouteFound); err != nil {
		logrus.Errorf("Failed to find %s route", "gitea-server")
		return reconcile.Result{}, err
	}

	giteaURL := "https://" + giteaRouteFound.Spec.Host

	for id := 1; id <= users; id++ {
		username := fmt.Sprintf("user%d", id)

		if result, err := createGitUser(workshop, username, giteaURL); err != nil {
			return result, err
		}
	}
	//Success
	return reconcile.Result{}, nil
}

func createGitUser(workshop *workshopv1.Workshop, username string, giteaURL string) (reconcile.Result, error) {

	var (
		openshiftUserPassword = workshop.Spec.User.Password
		err                   error
		httpResponse          *http.Response
		httpRequest           *http.Request
		requestURL            = giteaURL + "/user/sign_up"
		body                  = url.Values{}
		client                = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			// Do not follow Redirect
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	)

	body.Set("user_name", username)
	body.Set("email", username+"@none.com")
	body.Set("password", openshiftUserPassword)
	body.Set("retype", openshiftUserPassword)

	httpRequest, err = http.NewRequest("POST", requestURL, strings.NewReader(body.Encode()))
	httpRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Length", strconv.Itoa(len(body.Encode())))

	httpResponse, err = client.Do(httpRequest)
	if err != nil {
		return reconcile.Result{}, err
	}
	if httpResponse.StatusCode == http.StatusCreated {
		logrus.Infof("Created %s user in Gitea", username)
	}

	defer httpResponse.Body.Close()

	//Success
	return reconcile.Result{}, nil
}
