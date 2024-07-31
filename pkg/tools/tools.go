//go:build tools
// +build tools

package tools

import (
	_ "github.com/openshift/api/operator/v1/zz_generated.crd-manifests"
	_ "github.com/openshift/build-machinery-go"
	_ "sigs.k8s.io/kube-storage-version-migrator/manifests"
)
