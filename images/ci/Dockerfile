FROM registry.svc.ci.openshift.org/openshift/release:golang-1.12 AS builder
WORKDIR /go/src/github.com/openshift/cluster-kube-storage-version-migrator-operator
COPY . .

FROM registry.svc.ci.openshift.org/openshift/origin-v4.0:base
COPY --from=builder /go/src/github.com/openshift/cluster-kube-storage-version-migrator-operator/cluster-kube-storage-version-migrator-operator /usr/bin/
COPY manifests /manifests
# LABEL io.openshift.release.operator=true
