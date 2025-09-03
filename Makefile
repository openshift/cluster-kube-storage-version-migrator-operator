GO_REQUIRED_MIN_VERSION = 1.14

# Include the library makefiles
include $(addprefix ./vendor/github.com/openshift/build-machinery-go/make/, \
	golang.mk \
	targets/openshift/images.mk \
	targets/openshift/deps.mk \
	targets/openshift/operator/telepresence.mk \
	targets/openshift/operator/profile-manifests.mk \
)

# generate image targets
IMAGE_REGISTRY :=registry.svc.ci.openshift.org
$(call build-image,cluster-kube-storage-version-migrator-operator,$(IMAGE_REGISTRY)/ocp/4.4:cluster-kube-storage-version-migrator-operator,./images/ci/Dockerfile,.)

# include targets for profile manifest patches
$(call add-profile-manifests,manifests,./profile-patches,./manifests)

# exclude e2e test from unit tests
GO_TEST_PACKAGES :=./pkg/... ./cmd/...

# re-use test-unit target for e2e tests
.PHONY: test-e2e
test-e2e: GO_TEST_PACKAGES :=./test/e2e/...
test-e2e: test-unit

verify-vendor-manifests:
	bash -c 'diff -u <(grep -v include.release.openshift.io manifests/01_storage_migration_crd.yaml) vendor/sigs.k8s.io/kube-storage-version-migrator/manifests/storage_migration_crd.yaml'
	bash -c 'diff -u <(grep -v include.release.openshift.io manifests/01_storage_state_crd.yaml) vendor/sigs.k8s.io/kube-storage-version-migrator/manifests/storage_state_crd.yaml'
verify: verify-vendor-manifests
.PHONY: verify-vendor-manifests

# Configure the 'telepresence' target
# See vendor/github.com/openshift/build-machinery-go/scripts/run-telepresence.sh for usage and configuration details
export TP_DEPLOYMENT_YAML ?=./manifests/07_deployment.yaml
export TP_CMD_PATH ?=./cmd/cluster-kube-storage-version-migrator-operator

# Build the openshift-tests-extension binary
tests-ext-build:
	GOOS=$(if $(GOOS),$(GOOS),$(shell go env GOOS)) GOARCH=$(if $(GOARCH),$(GOARCH),$(shell go env GOARCH)) GO_COMPLIANCE_POLICY=exempt_all CGO_ENABLED=0 \
	go build -o cluster-kube-storage-version-migrator-operator-tests-ext \
	-ldflags "-X 'main.CommitFromGit=$(shell git rev-parse --short HEAD)' \
	-X 'main.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)' \
	-X 'main.GitTreeState=$(shell if git diff-index --quiet HEAD --; then echo clean; else echo dirty; fi)'" \
	./cmd/cluster-kube-storage-version-migrator-operator-tests-ext
.PHONY: tests-ext-build

# Update test metadata
tests-ext-update:
	./cluster-kube-storage-version-migrator-operator-tests-ext update
.PHONY: tests-ext-update
