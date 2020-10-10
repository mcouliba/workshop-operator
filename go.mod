module github.com/mcouliba/workshop-operator

go 1.15

require (
	github.com/argoproj-labs/argocd-operator v0.0.13
	github.com/eclipse/che-operator v0.0.0-20191211154745-df0be398efea
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openshift/api v3.9.1-0.20190916204813-cdbe64fb0c91+incompatible
	github.com/operator-framework/api v0.3.7-0.20200528122852-759ca0d84007
	github.com/prometheus/common v0.14.0
	github.com/sirupsen/logrus v1.6.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	k8s.io/api v0.18.9
	k8s.io/apiextensions-apiserver v0.18.6
	k8s.io/apimachinery v0.18.9
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.6.3
	sigs.k8s.io/yaml v1.2.0
)
