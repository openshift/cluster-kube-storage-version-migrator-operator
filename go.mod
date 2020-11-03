module github.com/openshift/cluster-kube-storage-version-migrator-operator

go 1.13

require (
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/openshift/api v0.0.0-20200929125329-c3027fc03b92
	github.com/openshift/build-machinery-go v0.0.0-20200819073603-48aa266c95f7
	github.com/openshift/client-go v0.0.0-20200827190008-3062137373b5
	github.com/openshift/library-go v0.0.0-20201102091359-c4fa0f5b3a08
	github.com/prometheus/client_golang v1.7.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v0.19.0
	k8s.io/component-base v0.19.0
	k8s.io/klog v1.0.0
)

replace (
	golang.org/x/text => golang.org/x/text v0.3.3
	k8s.io/api => k8s.io/api v0.19.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.0
	k8s.io/client-go => k8s.io/client-go v0.19.0
	k8s.io/component-base => k8s.io/component-base v0.19.0
)
