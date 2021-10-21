package e2e

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openshift/cluster-kube-storage-version-migrator-operator/pkg"
	library "github.com/openshift/cluster-kube-storage-version-migrator-operator/test/library"
)

func TestOperatorNamespace(t *testing.T) {
	kubeConfig, err := library.NewClientConfigForTest()
	if err != nil {
		t.Fatal(err)
	}
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		t.Fatal(err)
	}
	_, err = kubeClient.CoreV1().Namespaces().Get(context.Background(), pkg.OperatorNamespace, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestOperandNamespace(t *testing.T) {
	kubeConfig, err := library.NewClientConfigForTest()
	if err != nil {
		t.Fatal(err)
	}
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		t.Fatal(err)
	}
	_, err = kubeClient.CoreV1().Namespaces().Get(context.Background(), pkg.TargetNamespace, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
}
