# Include the library makefiles
include $(addprefix ./vendor/github.com/openshift/build-machinery-go/make/, \
	golang.mk \
	targets/openshift/bindata.mk \
	targets/openshift/images.mk \
	targets/openshift/deps.mk \
)

# generate bindata targets
$(call add-bindata,assets,./bindata/...,bindata,assets,pkg/operator/assets/bindata.go)

# generate image targets
IMAGE_REGISTRY :=registry.svc.ci.openshift.org
$(call build-image,cluster-kube-storage-version-migrator-operator,$(IMAGE_REGISTRY)/ocp/4.4:cluster-kube-storage-version-migrator-operator,./images/ci/Dockerfile,.)

# exclude e2e test from unit tests
GO_TEST_PACKAGES :=./pkg/... ./cmd/...

# re-use test-unit target for e2e tests
.PHONY: test-e2e
test-e2e: GO_TEST_PACKAGES :=./test/e2e/...
test-e2e: test-unit
