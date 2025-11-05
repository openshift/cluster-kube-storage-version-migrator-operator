# kube-storage-version-migrator operator

This operator manages the [kube-storage-version-migrator][migrator] and deploys the migration controller which:

* Processes migration requests by getting all objects of the migrated type and writing them back the API server without modification.  The purpose is to trigger the API server to encode the object in the latest storage version before storing it.

The trigger controller which performs storage migrations automatically is **not deployed** in OpenShift to keep migrations under control. It is up to operator owners to create [migration requests][].

[migrator]: https://github.com/openshift/kubernetes-kube-storage-version-migrator
[migration requests]:https://github.com/kubernetes-sigs/kube-storage-version-migrator/blob/60dee538334c2366994c2323c0db5db8ab4d2838/pkg/apis/migration/v1alpha1/types.go#L30

## Tests

The repository is compatible with the "OpenShift Tests Extension (OTE)" framework.

### Building the test binary
```bash
make build
```

### Running test suites and tests
```bash
# Run a specific test suite or test
./cluster-kube-storage-version-migrator-operator-tests-ext run-suite openshift/cluster-kube-storage-version-migrator-operator/all
./cluster-kube-storage-version-migrator-operator-tests-ext run-test "test-name"

# Run with JUnit output
./cluster-kube-storage-version-migrator-operator-tests-ext run-suite openshift/cluster-kube-storage-version-migrator-operator/all --junit-path=/tmp/junit-results/junit.xml
./cluster-kube-storage-version-migrator-operator-tests-ext run-test "test-name" --junit-path=/tmp/junit-results/junit.xml
```

### Listing available tests and suites
```bash
# List all test suites
./cluster-kube-storage-version-migrator-operator-tests-ext list suites

# List tests in a suite
./cluster-kube-storage-version-migrator-operator-tests-ext list tests --suite=openshift/cluster-kube-storage-version-migrator-operator/all

#for concurrency
./cluster-kube-storage-version-migrator-operator-tests-ext run-suite openshift/cluster-kube-storage-version-migrator-operator/all -c 1
```
