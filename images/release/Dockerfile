FROM openshift/golang-builder:1.12 AS builder
WORKDIR /go/src/github.com/openshift/cluster-kube-storage-version-migrator-operator
COPY . .

FROM openshift/ose-base:v4.3
COPY --from=builder /go/src/github.com/openshift/cluster-kube-storage-version-migrator-operator/cluster-kube-storage-version-migrator-operator /usr/bin/
COPY manifests /manifests
# LABEL io.openshift.release.operator=true
