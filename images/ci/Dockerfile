FROM registry.svc.ci.openshift.org/openshift/release:golang-1.13 AS builder
WORKDIR /go/src/github.com/openshift/cluster-kube-storage-version-migrator-operator
COPY . .
RUN make build --warn-undefined-variables

FROM registry.svc.ci.openshift.org/openshift/origin-v4.4:base
COPY --from=builder /go/src/github.com/openshift/cluster-kube-storage-version-migrator-operator/cluster-kube-storage-version-migrator-operator /usr/bin/
COPY manifests /manifests
COPY vendor/github.com/openshift/api/operator/v1/*_kube-storage-version-migrator-operator_*.yaml* /manifests
LABEL io.openshift.release.operator=true
