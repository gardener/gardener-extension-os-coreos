module github.com/gardener/gardener-extension-os-coreos

go 1.16

require (
	github.com/gardener/gardener v1.25.0
	github.com/go-logr/logr v0.3.0
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.5
	github.com/spf13/cobra v1.1.1
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/component-base v0.20.7
	sigs.k8s.io/controller-runtime v0.8.3
)

replace (
	k8s.io/api => k8s.io/api v0.20.7
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.7
	k8s.io/client-go => k8s.io/client-go v0.20.7
	k8s.io/code-generator => k8s.io/code-generator v0.20.7
	k8s.io/component-base => k8s.io/component-base v0.20.7
	k8s.io/helm => k8s.io/helm v2.13.1+incompatible
)
