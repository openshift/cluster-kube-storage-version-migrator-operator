module github.com/openshift/cluster-kube-storage-version-migrator-operator

go 1.16

require (
	github.com/google/go-cmp v0.5.5
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/openshift/api v0.0.0-20211209135129-c58d9f695577
	github.com/openshift/build-machinery-go v0.0.0-20210922160744-a9caf93aef90
	github.com/openshift/client-go v0.0.0-20211209144617-7385dd6338e3
	github.com/openshift/library-go v0.0.0-20211214183058-58531ccbde67
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.23.0
	k8s.io/apimachinery v0.23.0
	k8s.io/client-go v0.23.0
	k8s.io/component-base v0.23.0
	sigs.k8s.io/kube-storage-version-migrator v0.0.5-0.20210421184352-acdee30ced21
)

replace sigs.k8s.io/kube-storage-version-migrator => github.com/openshift/kube-storage-version-migrator v0.0.3-0.20210503105529-901a6d221d1c
