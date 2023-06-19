# kube-storage-version-migrator operator

This operator manages the [kube-storage-version-migrator][migrator] and deploys the migration controller which:

* Processes migration requests by getting all objects of the migrated type and writing them back the API server without modification.  The purpose is to trigger the API server to encode the object in the latest storage version before storing it.

The trigger controller which performs storage migrations automatically is **not deployed** in OpenShift to keep migrations under control. It is up to operator owners to create [migration requests][].

[migrator]: https://github.com/openshift/kubernetes-kube-storage-version-migrator
[migration requests]:https://github.com/kubernetes-sigs/kube-storage-version-migrator/blob/60dee538334c2366994c2323c0db5db8ab4d2838/pkg/apis/migration/v1alpha1/types.go#L30
