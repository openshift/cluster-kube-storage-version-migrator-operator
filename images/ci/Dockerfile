FROM registry.ci.openshift.org/ocp/builder:rhel-9-golang-1.22-openshift-4.18 AS builder
WORKDIR /go/src/github.com/openshift/cluster-kube-storage-version-migrator-operator
COPY . .
ENV GO_PACKAGE github.com/openshift/cluster-kube-storage-version-migrator-operator
RUN make build --warn-undefined-variables

FROM registry.ci.openshift.org/ocp/4.18:base-rhel9
COPY --from=builder /go/src/github.com/openshift/cluster-kube-storage-version-migrator-operator/cluster-kube-storage-version-migrator-operator /usr/bin/
COPY manifests /manifests
COPY vendor/github.com/openshift/api/operator/v1/zz_generated.crd-manifests/*_kubestorageversionmigrators.crd.yaml /manifests
LABEL io.openshift.release.operator=true
