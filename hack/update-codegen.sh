#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

export GOPATH=$(go env GOPATH)

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_ROOT="${SCRIPT_DIR}"/..

# setup virtual GOPATH
# k8s.io/code-generator does not work outside GOPATH, see https://github.com/kubernetes/kubernetes/issues/86753.
source "$GARDENER_HACK_DIR"/vgopath-setup.sh

# fetch code-generator module to execute the scripts from the modcache (we don't vendor here)
CODE_GENERATOR_DIR="$(go list -m -tags tools -f '{{ .Dir }}' k8s.io/code-generator)"

# We need to explicitly pass GO111MODULE=off to k8s.io/code-generator as it is significantly slower otherwise,
# see https://github.com/kubernetes/code-generator/issues/100.
export GO111MODULE=off

source "${CODE_GENERATOR_DIR}/kube_codegen.sh"

kube::codegen::gen_helpers \
  --boilerplate "${GARDENER_HACK_DIR}/LICENSE_BOILERPLATE.txt" \
  "${PROJECT_ROOT}/pkg/"