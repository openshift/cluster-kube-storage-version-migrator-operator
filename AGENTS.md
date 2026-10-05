# AI Agent Instructions for cluster-kube-storage-version-migrator-operator

> Also read `ARCHITECTURE.md` for reconciliation flow and controller details.

## What This Repo Is

This is an OpenShift operator that manages the singleton kube-storage-version-migrator operand. The operand re-writes Kubernetes objects in etcd to ensure they are encoded at the latest storage version after API server upgrades. The operator handles operand deployment, RBAC, network policies, and reports status via the `kube-storage-version-migrator` ClusterOperator.

## Repository Layout

```text
cmd/
  cluster-kube-storage-version-migrator-operator/         # Operator binary
  cluster-kube-storage-version-migrator-operator-tests-ext/ # OTE test binary
pkg/
  operator/starter.go                   # Controller wiring (RunOperator)
  operator/deploymentcontroller/        # Operand deployment reconciliation
  operator/staticconditionscontroller/   # DefaultUpgradeable=True
  const.go                              # Namespace constants
bindata/kube-storage-version-migrator/  # Operand static resources (embedded)
manifests/                              # CVO install payload for the operator
test/e2e/                               # Ginkgo E2E tests (OTE-registered)
```

## Build and Test Commands

```bash
make build       # Compile operator + OTE test binaries
make test-unit   # Unit tests (./pkg/... ./cmd/...)
make verify      # gofmt, go vet, vendor manifest verification
```

E2E tests require a real OpenShift cluster and run via the OTE binary:

```bash
./cluster-kube-storage-version-migrator-operator-tests-ext run-suite \
  --suite openshift/cluster-kube-storage-version-migrator-operator/operator/parallel
```

## Critical Rules

1. **Do not edit vendored files.** The `vendor/` directory is managed by `go mod tidy && go mod vendor`. Never hand-edit anything under `vendor/`.
2. **Do not edit generated files.** Files matching `zz_generated.*` are generated upstream.
3. **Use `make build`, not `go build`.** The Makefile injects version info via ldflags and places binaries correctly.
4. **After modifying `go.mod`:** always run `go mod tidy && go mod vendor` and commit the vendor changes.
5. **CRD manifests must match vendor.** Run `make verify` — it diffs `manifests/01_*.yaml` against vendored copies.

## Key Patterns

- **Static + dynamic resource split**: `bindata/` resources are applied by `StaticResourceController`; the operand Deployment is reconciled by `DeploymentController` with hooks.
- **HyperShift awareness**: Deployment controller adjusts replica count and node selector based on `Infrastructure` topology mode.
- **library-go delegation**: Most reconciliation logic (status, stale conditions, log level, static resources) comes from `openshift/library-go`. Local code is hooks and wiring.
- **Single operator CR**: `KubeStorageVersionMigrator` at `operator.openshift.io/v1` (named `cluster`). Spec drives log level and management state.
- **Operand image injection**: `${IMAGE}` placeholder in bindata deployment is replaced at runtime from `IMAGE` env var.

## What NOT to Do

- CRD definitions are maintained in the upstream `kube-storage-version-migrator` fork — do not edit them here directly.
- OWNERS and OWNERS_ALIASES files are not to be modified without team consensus.
- E2E tests require a full OpenShift cluster — do not run them on Kind or Minikube.
- When adding cloud-specific or platform-specific behavior, gate it on `Infrastructure` topology, not hardcoded checks.
- The trigger controller is intentionally **not deployed** in OpenShift. Do not add automatic migration triggering logic.

## Test Suites (OTE)

The OTE binary registers two suites:

| Suite | Parallelism | Qualifier |
|-------|-------------|-----------|
| `.../operator/disruptive` | 1 | `[Disruptive] && [Serial]` |
| `.../operator/parallel` | default | `![Serial] && ![Disruptive]` |

Tag tests with `[Serial]` if they mutate shared state, and `[Disruptive]` if they delete or break cluster resources.
