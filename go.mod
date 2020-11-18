module github.com/openshift/cluster-kube-storage-version-migrator-operator

go 1.13

require (
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/go-logr/logr v0.3.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/openshift/api v0.0.0-20201117184740-859beeffd973
	github.com/openshift/build-machinery-go v0.0.0-20200917070002-f171684f77ab
	github.com/openshift/client-go v0.0.0-20201020074620-f8fd44879f7c
	github.com/openshift/library-go v0.0.0-20201102091359-c4fa0f5b3a08
	github.com/prometheus/client_golang v1.7.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b // indirect
	golang.org/x/text v0.3.4 // indirect
	k8s.io/api v0.19.4
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v0.19.2
	k8s.io/component-base v0.19.2
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.4.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.0.2 // indirect
)

replace (
	golang.org/x/text => golang.org/x/text v0.3.3
	k8s.io/api => k8s.io/api v0.19.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.0
	k8s.io/client-go => k8s.io/client-go v0.19.0
	k8s.io/component-base => k8s.io/component-base v0.19.0
)
