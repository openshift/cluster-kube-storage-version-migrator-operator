module github.com/openshift/cluster-kube-storage-version-migrator-operator

go 1.16

require (
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/openshift/api v0.0.0-20210726144523-6fcabc0010ca
	github.com/openshift/build-machinery-go v0.0.0-20210806203541-4ea9b6da3a37
	github.com/openshift/client-go v0.0.0-20210521082421-73d9475a9142
	github.com/openshift/library-go v0.0.0-20210728105326-4a78e9dddb00
	github.com/prometheus/client_golang v1.7.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
	k8s.io/component-base v0.21.1
	k8s.io/klog/v2 v2.8.0
	sigs.k8s.io/kube-storage-version-migrator v0.0.5-0.20210421184352-acdee30ced21
)

replace (
	// points to temporary-watch-reduction-patch-1.21 to pick up k/k/pull/101102 - please remove it once the pr merges and a new Z release is cut
	k8s.io/apiserver => github.com/openshift/kubernetes-apiserver v0.0.0-20210419140141-620426e63a99
	sigs.k8s.io/kube-storage-version-migrator => github.com/openshift/kube-storage-version-migrator v0.0.3-0.20210503105529-901a6d221d1c
)
