package nexus

import (
	workshopv1 "github.com/mcouliba/workshop-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewCustomResource create a Custom Resource
func NewCustomResource(workshop *workshopv1.Workshop, scheme *runtime.Scheme,
	name string, namespace string, labels map[string]string) *Nexus {

	cr := &Nexus{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: NexusSpec{
			NexusVolumeSize:    "5Gi",
			NexusSSL:           true,
			NexusImageTag:      "3.18.1-01-ubi-3",
			NexusCPURequest:    1,
			NexusCPULimit:      2,
			NexusMemoryRequest: "2Gi",
			NexusMemoryLimit:   "2Gi",
			NexusReposMavenProxy: []NexusReposMavenProxySpec{
				{
					Name:         "maven-central",
					RemoteURL:    "https://repo1.maven.org/maven2/",
					LayoutPolicy: "permissive",
				},
				{
					Name:         "redhat-ga",
					RemoteURL:    "https://maven.repository.redhat.com/ga/",
					LayoutPolicy: "permissive",
				},
				{
					Name:         "jboss",
					RemoteURL:    "https://repository.jboss.org/nexus/content/groups/public",
					LayoutPolicy: "permissive",
				},
			},
			NexusReposMavenHosted: []NexusReposMavenHostedSpec{
				{
					Name:          "releases",
					VersionPolicy: "release",
					WritePolicy:   "allow_once",
				},
			},
			NexusReposMavenGroup: []NexusReposMavenGroupSpec{
				{
					Name:        "maven-all-public",
					MemberRepos: []string{"maven-central", "redhat-ga", "jboss"},
				},
			},
			NexusReposDockerHosted: []NexusReposDockerHostedSpec{
				{
					Name:      "docker",
					HttpPort:  5000,
					V1Enabled: true,
				},
			},
			NexusReposNpmProxy: []NexusReposNpmProxySpec{
				{
					Name:      "npm",
					RemoteURL: "https://registry.npmjs.org",
				},
			},
			NexusReposNpmGroup: []NexusReposNpmGroupSpec{
				{
					Name:        "npm-all",
					MemberRepos: []string{"npm"},
				},
			},
		},
	}

	// Set Workshop instance as the owner and controller
	ctrl.SetControllerReference(workshop, cr, scheme)

	return cr
}
