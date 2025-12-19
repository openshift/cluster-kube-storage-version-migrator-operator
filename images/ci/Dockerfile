FROM registry.ci.openshift.org/ocp/builder:rhel-9-golang-1.24-openshift-4.22 AS builder
WORKDIR /go/src/github.com/openshift/cluster-kube-storage-version-migrator-operator
COPY . .
ENV GO_PACKAGE github.com/openshift/cluster-kube-storage-version-migrator-operator
RUN make build --warn-undefined-variables \
    && gzip cluster-kube-storage-version-migrator-operator-tests-ext

FROM registry.ci.openshift.org/ocp/4.22:base-rhel9
COPY --from=builder /go/src/github.com/openshift/cluster-kube-storage-version-migrator-operator/cluster-kube-storage-version-migrator-operator /usr/bin/
COPY --from=builder /go/src/github.com/openshift/cluster-kube-storage-version-migrator-operator/cluster-kube-storage-version-migrator-operator-tests-ext.gz /usr/bin/
COPY manifests /manifests
COPY vendor/github.com/openshift/api/operator/v1/zz_generated.crd-manifests/*_kubestorageversionmigrators.crd.yaml /manifests
LABEL io.openshift.release.operator=true
