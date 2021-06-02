# kube-storage-version-migrator operator

This operator manages the [kube-storage-version-migrator][migrator], which:

* Detects changes of the default storage version of a resource type by polling the API server's discovery document.
* Creates migration requests for resource types whose storage version changes.
* Processes migration requests by geting all objects of the migrated type and writing them back the API server without modification.  The purpose is to trigger the API server to encode the object in the latest storage version before storing it.

[migrator]: https://github.com/openshift/kubernetes-kube-storage-version-migrator
