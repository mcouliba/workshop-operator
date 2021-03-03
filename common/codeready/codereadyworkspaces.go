package codeready

import (
	che "github.com/eclipse/che-operator/pkg/apis/org/v1"
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type codeReadyUser struct {
	Username    string       `json:"username"`
	Enabled     bool         `json:"enabled"`
	Email       string       `json:"email"`
	Credentials []credential `json:"credentials"`
	ClientRoles clientRoles  `json:"clientRoles"`
}

type credential struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type clientRoles struct {
	RealmManagement []string `json:"realm-management"`
}

// NewCustomResource creates a Custom Resource
func NewCustomResource(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string) *che.CheCluster {

	pluginRegistryImage := workshop.Spec.Infrastructure.CodeReadyWorkspace.PluginRegistryImage.Name +
		":" + workshop.Spec.Infrastructure.CodeReadyWorkspace.PluginRegistryImage.Tag

	cr := &che.CheCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CheCluster",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: che.CheClusterSpec{
			Server: che.CheClusterSpecServer{
				CheImageTag: "",
				CheFlavor:   "codeready",
				CustomCheProperties: map[string]string{
					"CHE_INFRA_KUBERNETES_NAMESPACE_DEFAULT": "<username>-workspace",
					"CHE_LIMITS_USER_WORKSPACES_RUN_COUNT":   "2",
					"CHE_LIMITS_WORKSPACE_IDLE_TIMEOUT":      "0",
				},
				DevfileRegistryImage: "",
				PluginRegistryImage:  pluginRegistryImage,
				TlsSupport:           true,
				SelfSignedCert:       false,
			},
			Database: che.CheClusterSpecDB{
				ExternalDb:          false,
				ChePostgresHostName: "",
				ChePostgresPort:     "",
				ChePostgresUser:     "",
				ChePostgresPassword: "",
				ChePostgresDb:       "",
			},
			Auth: che.CheClusterSpecAuth{
				OpenShiftoAuth:                workshop.Spec.Infrastructure.CodeReadyWorkspace.OpenshiftOAuth,
				IdentityProviderImage:         "",
				ExternalIdentityProvider:      false,
				IdentityProviderURL:           "",
				IdentityProviderRealm:         "",
				IdentityProviderClientId:      "",
				IdentityProviderAdminUserName: "admin",
				IdentityProviderPassword:      "admin",
			},
			Storage: che.CheClusterSpecStorage{
				PvcStrategy:       "per-workspace",
				PvcClaimSize:      "1Gi",
				PreCreateSubPaths: true,
			},
		},
	}

	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, cr, scheme)
	return cr
}

// NewUser creates a user
func NewUser(username string, password string) *codeReadyUser {
	return &codeReadyUser{
		Username: username,
		Enabled:  true,
		Email:    username + "@none.com",
		Credentials: []credential{
			{
				Type:  "password",
				Value: password,
			},
		},
		ClientRoles: clientRoles{
			RealmManagement: []string{
				"user",
			},
		},
	}
}
