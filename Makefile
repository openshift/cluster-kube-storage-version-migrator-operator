all: build
.PHONY: all

# Include the library makefile
include $(addprefix ./vendor/github.com/openshift/library-go/alpha-build-machinery/make/, \
	golang.mk \
	targets/openshift/bindata.mk \
	targets/openshift/images.mk \
)

# generate bindata targets
$(call add-bindata,assets,./bindata/...,bindata,assets,pkg/operator/assets/bindata.go)

# generate image targets
IMAGE_REGISTRY :=registry.svc.ci.openshift.org
$(call build-image,cluster-kube-storage-version-migrator-operator,$(IMAGE_REGISTRY)/ocp/4.3:cluster-kube-storage-version-migrator-operator,./images/ci/Dockerfile,.)
