module github.com/openshift/cluster-kube-storage-version-migrator-operator

go 1.13

require (
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/openshift/api v0.0.0-20200422085536-f24cbf292bdd
	github.com/openshift/build-machinery-go v0.0.0-20200512074546-3744767c4131
	github.com/openshift/client-go v0.0.0-20200422192633-6f6c07fc2a70
	github.com/openshift/library-go v0.0.0-20200423145702-b0e5b39cd9e7
	github.com/prometheus/client_golang v1.2.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v0.18.2
	k8s.io/component-base v0.18.2
	k8s.io/klog v1.0.0
)

replace (
	golang.org/x/text => golang.org/x/text v0.3.3
	k8s.io/api => k8s.io/api v0.18.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.2
	k8s.io/client-go => k8s.io/client-go v0.18.2
	k8s.io/component-base => k8s.io/component-base v0.18.2
)
