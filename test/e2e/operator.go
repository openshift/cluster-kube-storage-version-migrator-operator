package e2e

import (
	"context"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/cluster-kube-storage-version-migrator-operator/pkg"
)

var _ = g.Describe("[sig-api-machinery] cluster-kube-storage-version-migrator-operator", func() {

	g.It("should have the operator namespace", func() {
		kubeClient := newKubeClient()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, err := kubeClient.CoreV1().Namespaces().Get(ctx, pkg.OperatorNamespace, metav1.GetOptions{})
		o.Expect(err).NotTo(o.HaveOccurred(), "operator namespace %s should exist", pkg.OperatorNamespace)
	})

	g.It("should have the operand namespace", func() {
		kubeClient := newKubeClient()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, err := kubeClient.CoreV1().Namespaces().Get(ctx, pkg.TargetNamespace, metav1.GetOptions{})
		o.Expect(err).NotTo(o.HaveOccurred(), "operand namespace %s should exist", pkg.TargetNamespace)
	})
})
