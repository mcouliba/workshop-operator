module github.com/mcouliba/workshop-operator

go 1.15

require (
	github.com/argoproj-labs/argocd-operator v0.0.13
	github.com/eclipse/che-operator v0.0.0-20191211154745-df0be398efea
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.9 // indirect
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openshift/api v3.9.1-0.20190916204813-cdbe64fb0c91+incompatible
	github.com/operator-framework/api v0.3.17
	github.com/prometheus/common v0.14.0
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0
	k8s.io/api v0.18.9
	k8s.io/apiextensions-apiserver v0.18.9
	k8s.io/apimachinery v0.18.9
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.6.3
	sigs.k8s.io/yaml v1.2.0
)

replace k8s.io/client-go => k8s.io/client-go v0.18.9
