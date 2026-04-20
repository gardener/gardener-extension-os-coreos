// +k8s:deepcopy-gen=package
// +k8s:defaulter-gen=TypeMeta

//go:generate crd-ref-docs --source-path=. --config=../../../../hack/api-reference/config.yaml --renderer=markdown --templates-dir=$GARDENER_HACK_DIR/api-reference/template --log-level=ERROR --output-path=../../../../hack/api-reference/config.md

// Package v1alpha1 contains the API for configuring the os-coreos extension.
// +groupName=config.coreos.os.extensions.gardener.cloud
package v1alpha1 // import "github.com/gardener/gardener-extension-os-coreos/pkg/controller/config/v1alpha1"
