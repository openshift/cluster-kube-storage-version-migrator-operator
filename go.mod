module github.com/openshift/cluster-kube-storage-version-migrator-operator

go 1.13

require (
	github.com/go-bindata/go-bindata v3.1.1+incompatible
	github.com/openshift/api v0.0.0-20191217141120-791af96035a5
	github.com/openshift/build-machinery-go v0.0.0-20200424080330-082bf86082cc
	github.com/openshift/client-go v0.0.0-20191216194936-57f413491e9e
	github.com/openshift/library-go v0.0.0-20191218095328-1c12909e5923
	github.com/prometheus/client_golang v1.2.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v0.17.0
	k8s.io/component-base v0.17.0
	k8s.io/klog v1.0.0
)

replace (
	github.com/openshift/api => github.com/openshift/api v0.0.0-20191217141120-791af96035a5
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20191216194936-57f413491e9e
	github.com/openshift/library-go => github.com/openshift/library-go v0.0.0-20191218095328-1c12909e5923
	k8s.io/api => k8s.io/api v0.17.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.0
	k8s.io/client-go => k8s.io/client-go v0.17.0
	k8s.io/component-base => k8s.io/component-base v0.17.0
)
