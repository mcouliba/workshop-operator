/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WorkshopSpec defines the desired state of Workshop
type WorkshopSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	User           UserSpec           `json:"user"`
	Source         SourceSpec         `json:"source"`
	Infrastructure InfrastructureSpec `json:"infrastructure"`
}

// UserSpec ...
type UserSpec struct {
	Number   int    `json:"number"`
	Password string `json:"password"`
}

// SourceSpec ...
type SourceSpec struct {
	GitURL    string `json:"gitURL"`
	GitBranch string `json:"gitBranch"`
}

// InfrastructureSpec ...
type InfrastructureSpec struct {
	CertManager        CertManagerSpec        `json:"certManager,omitempty"`
	CodeReadyWorkspace CodeReadyWorkspaceSpec `json:"codeReadyWorkspace,omitempty"`
	Gitea              GiteaSpec              `json:"gitea,omitempty"`
	GitOps             GitOpsSpec             `json:"gitops,omitempty"`
	Guide              GuideSpec              `json:"guide,omitempty"`
	IstioWorkspace     IstioWorkspaceSpec     `json:"istioWorkspace,omitempty"`
	Nexus              NexusSpec              `json:"nexus,omitempty"`
	Pipeline           PipelineSpec           `json:"pipeline,omitempty"`
	Project            ProjectSpec            `json:"project,omitempty"`
	ServiceMesh        ServiceMeshSpec        `json:"serviceMesh,omitempty"`
	Serverless         ServerlessSpec         `json:"serverless,omitempty"`
	Vault              VaultSpec              `json:"vault,omitempty"`
}

// BookbagSpec ...
type BookbagSpec struct {
	Enabled bool      `json:"enabled"`
	Image   ImageSpec `json:"image"`
}

// CertManagerSpec ...
type CertManagerSpec struct {
	Enabled     bool            `json:"enabled"`
	OperatorHub OperatorHubSpec `json:"operatorHub"`
}

// GiteaSpec ...
type GiteaSpec struct {
	Enabled bool      `json:"enabled"`
	Image   ImageSpec `json:"image"`
}

// GitOpsSpec ...
type GitOpsSpec struct {
	Enabled     bool            `json:"enabled"`
	OperatorHub OperatorHubSpec `json:"operatorHub"`
}

// GuideSpec ...
type GuideSpec struct {
	Bookbag  BookbagSpec  `json:"bookbag,omitempty"`
	Scholars ScholarsSpec `json:"scholars,omitempty"`
}

// NexusSpec ...
type NexusSpec struct {
	Enabled bool `json:"enabled"`
}

// PipelineSpec ...
type PipelineSpec struct {
	Enabled     bool            `json:"enabled"`
	OperatorHub OperatorHubSpec `json:"operatorHub"`
}

// ProjectSpec ...
type ProjectSpec struct {
	Enabled     bool   `json:"enabled"`
	StagingName string `json:"stagingName"`
}

// ScholarsSpec ...
type ScholarsSpec struct {
	Enabled  bool              `json:"enabled"`
	GuideURL map[string]string `json:"guideURL"`
}

// ServiceMeshSpec ...
type ServiceMeshSpec struct {
	Enabled                  bool            `json:"enabled"`
	ServiceMeshOperatorHub   OperatorHubSpec `json:"serviceMeshOperatorHub"`
	ElasticSearchOperatorHub OperatorHubSpec `json:"elasticSearchOperatorHub"`
	JaegerOperatorHub        OperatorHubSpec `json:"jaegerOperatorHub"`
	KialiOperatorHub         OperatorHubSpec `json:"kialiOperatorHub"`
}

// ServerlessSpec ...
type ServerlessSpec struct {
	Enabled     bool            `json:"enabled"`
	OperatorHub OperatorHubSpec `json:"operatorHub"`
}

// CodeReadyWorkspaceSpec ...
type CodeReadyWorkspaceSpec struct {
	Enabled             bool            `json:"enabled"`
	OperatorHub         OperatorHubSpec `json:"operatorHub"`
	OpenshiftOAuth      bool            `json:"openshiftOAuth"`
	PluginRegistryImage ImageSpec       `json:"pluginRegistryImage,omitempty"`
}

// IstioWorkspaceSpec ...
type IstioWorkspaceSpec struct {
	Enabled     bool            `json:"enabled"`
	OperatorHub OperatorHubSpec `json:"operatorHub"`
}

// OperatorHubSpec ...
type OperatorHubSpec struct {
	Channel               string `json:"channel"`
	ClusterServiceVersion string `json:"clusterServiceVersion,omitempty"`
}

// ImageSpec ...
type ImageSpec struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

// VaultSpec ...
type VaultSpec struct {
	Enabled            bool      `json:"enabled"`
	Image              ImageSpec `json:"image"`
	AgentInjectorImage ImageSpec `json:"agentInjectorImage"`
}

// WorkshopStatus defines the observed state of Workshop
type WorkshopStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Bookbag              string `json:"bookbag"`
	CertManager          string `json:"certManager"`
	CodeReadyWorkspace   string `json:"codeReadyWorkspace"`
	Gitea                string `json:"gitea"`
	GitOps               string `json:"gitops"`
	IstioWorkspace       string `json:"istioWorkspace"`
	Nexus                string `json:"nexus"`
	Pipeline             string `json:"pipeline"`
	Project              string `json:"project"`
	ServiceMesh          string `json:"serviceMesh"`
	Serverless           string `json:"serverless"`
	UsernameDistribution string `json:"usernameDistribution"`
	Vault                string `json:"vault"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Workshop is the Schema for the workshops API
type Workshop struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkshopSpec   `json:"spec,omitempty"`
	Status WorkshopStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WorkshopList contains a list of Workshop
type WorkshopList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workshop `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Workshop{}, &WorkshopList{})
}
