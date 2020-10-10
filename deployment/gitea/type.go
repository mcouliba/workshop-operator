package gitea

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Gitea struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GiteaSpec `json:"spec,omitempty"`
}

type GiteaSpec struct {
	GiteaVolumeSize      string `json:"giteaVolumeSize"`
	GiteaSsl             bool   `json:"giteaSsl"`
	GiteaServiceName     string `json:"giteaServiceName,omitempty"`
	PostgresqlVolumeSize string `json:"postgresqlVolumeSize"`
}

type GiteaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Gitea `json:"items"`
}
