package e2e

import (
	"context"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openshift/cluster-kube-storage-version-migrator-operator/pkg"
	testlibrary "github.com/openshift/cluster-kube-storage-version-migrator-operator/test/library"
)

const (
	allowMigratorPolicyName = "allow-migrator"
	allowOperatorPolicyName = "allow-operator"
	defaultDenyPolicyName   = "default-deny"
	reconcileTimeout        = 5 * time.Minute
)

func newKubeClient() kubernetes.Interface {
	kubeConfig, err := testlibrary.NewClientConfigForTest()
	o.Expect(err).NotTo(o.HaveOccurred())
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	o.Expect(err).NotTo(o.HaveOccurred())
	return kubeClient
}

var _ = g.Describe("[sig-api-machinery] cluster-kube-storage-version-migrator-operator", func() {

	g.It("[NetworkPolicy] should ensure NetworkPolicies are defined in the operand namespace", func() {
		t := g.GinkgoTB()
		ctx := context.Background()
		kubeClient := newKubeClient()

		policies, err := kubeClient.NetworkingV1().NetworkPolicies(pkg.TargetNamespace).List(ctx, metav1.ListOptions{})
		o.Expect(err).NotTo(o.HaveOccurred())
		LogPolicyNames(t, pkg.TargetNamespace, policies.Items)

		g.By("Checking default-deny policy exists")
		o.Expect(HasDefaultDeny(policies.Items)).To(o.BeTrue(),
			"expected a default-deny policy in %s", pkg.TargetNamespace)

		g.By("Checking allow-migrator policy exists")
		found := false
		for _, p := range policies.Items {
			if p.Name == allowMigratorPolicyName {
				found = true
				break
			}
		}
		o.Expect(found).To(o.BeTrue(),
			"expected %s policy in %s", allowMigratorPolicyName, pkg.TargetNamespace)
	})

	g.It("[NetworkPolicy] should validate default-deny policy structure", func() {
		t := g.GinkgoTB()
		ctx := context.Background()
		kubeClient := newKubeClient()

		policy := GetNetworkPolicy(t, ctx, kubeClient, pkg.TargetNamespace, defaultDenyPolicyName)
		LogNetworkPolicySummary(t, defaultDenyPolicyName, policy)
		LogNetworkPolicyDetails(t, defaultDenyPolicyName, policy)

		g.By("Validating pod selector is empty (applies to all pods)")
		RequireEmptyPodSelector(t, policy)

		g.By("Validating policy types include both Ingress and Egress")
		o.Expect(policy.Spec.PolicyTypes).To(o.ContainElement(networkingv1.PolicyTypeIngress))
		o.Expect(policy.Spec.PolicyTypes).To(o.ContainElement(networkingv1.PolicyTypeEgress))

		g.By("Validating no allow rules are present")
		o.Expect(policy.Spec.Ingress).To(o.BeEmpty(), "default-deny should have no ingress rules")
		o.Expect(policy.Spec.Egress).To(o.BeEmpty(), "default-deny should have no egress rules")
	})

	g.It("[NetworkPolicy] should validate allow-migrator policy structure", func() {
		t := g.GinkgoTB()
		ctx := context.Background()
		kubeClient := newKubeClient()

		policy := GetNetworkPolicy(t, ctx, kubeClient, pkg.TargetNamespace, allowMigratorPolicyName)
		LogNetworkPolicySummary(t, allowMigratorPolicyName, policy)
		LogNetworkPolicyDetails(t, allowMigratorPolicyName, policy)

		g.By("Validating pod selector targets migrator pods")
		RequirePodSelectorLabel(t, policy, "app", "migrator")

		g.By("Validating unrestricted egress")
		RequireUnrestrictedEgress(t, policy)

		g.By("Validating policy type is Egress only")
		o.Expect(policy.Spec.PolicyTypes).To(o.ContainElement(networkingv1.PolicyTypeEgress))
		o.Expect(policy.Spec.PolicyTypes).NotTo(o.ContainElement(networkingv1.PolicyTypeIngress),
			"allow-migrator should not include Ingress policy type")

		g.By("Validating no ingress rules are present")
		o.Expect(policy.Spec.Ingress).To(o.BeEmpty(), "allow-migrator should have no ingress rules")
	})

	g.It("[NetworkPolicy] should verify migrator egress connectivity", func() {
		t := g.GinkgoTB()
		ctx := context.Background()
		kubeClient := newKubeClient()

		migratorLabels := map[string]string{"app": "migrator"}

		g.By("Verifying egress to API server on port 443")
		kubeSvc, err := kubeClient.CoreV1().Services("default").Get(ctx, "kubernetes", metav1.GetOptions{})
		o.Expect(err).NotTo(o.HaveOccurred())
		ExpectConnectivity(ctx, t, kubeClient, pkg.TargetNamespace, migratorLabels, []string{kubeSvc.Spec.ClusterIP}, 443, true)

		g.By("Verifying egress to DNS on port 53")
		dnsSvc, err := kubeClient.CoreV1().Services("openshift-dns").Get(ctx, "dns-default", metav1.GetOptions{})
		o.Expect(err).NotTo(o.HaveOccurred())
		ExpectConnectivity(ctx, t, kubeClient, pkg.TargetNamespace, migratorLabels, []string{dnsSvc.Spec.ClusterIP}, 53, true)
	})

	g.It("[NetworkPolicy][Serial][Disruptive] should deny ingress to operand namespace pods", func() {
		t := g.GinkgoTB()
		ctx := context.Background()
		kubeClient := newKubeClient()

		g.By("Creating a server pod in the operand namespace")
		serverLabels := map[string]string{"test": "np-ingress-deny"}
		serverIPs, cleanup := CreateServerPod(ctx, t, kubeClient, pkg.TargetNamespace, "np-deny-server", serverLabels, 8080)
		defer cleanup()

		g.By("Verifying ingress from external namespace is denied")
		clientLabels := map[string]string{"test": "np-ingress-deny-client"}
		ExpectConnectivity(ctx, t, kubeClient, "default", clientLabels, serverIPs, 8080, false)
	})

	g.It("[NetworkPolicy][Serial][Disruptive] should restore operand NetworkPolicies after delete[Timeout:15m]", func() {
		t := g.GinkgoTB()
		ctx := context.Background()
		kubeClient := newKubeClient()

		g.By("Capturing expected policies")
		expectedAllow := GetNetworkPolicy(t, ctx, kubeClient, pkg.TargetNamespace, allowMigratorPolicyName)
		expectedDeny := GetNetworkPolicy(t, ctx, kubeClient, pkg.TargetNamespace, defaultDenyPolicyName)

		g.By("Deleting allow-migrator and waiting for restoration")
		RestoreNetworkPolicy(t, ctx, kubeClient, expectedAllow, reconcileTimeout)

		g.By("Deleting default-deny and waiting for restoration")
		RestoreNetworkPolicy(t, ctx, kubeClient, expectedDeny, reconcileTimeout)

		g.By("Checking NetworkPolicy-related events")
		LogNetworkPolicyEvents(t, ctx, kubeClient,
			[]string{pkg.OperatorNamespace, pkg.TargetNamespace}, allowMigratorPolicyName)
		LogNetworkPolicyEvents(t, ctx, kubeClient,
			[]string{pkg.OperatorNamespace, pkg.TargetNamespace}, defaultDenyPolicyName)
	})

	g.It("[NetworkPolicy][Serial][Disruptive] should restore operand NetworkPolicies after mutation[Timeout:15m]", func() {
		t := g.GinkgoTB()
		ctx := context.Background()
		kubeClient := newKubeClient()

		g.By("Mutating allow-migrator and waiting for reconciliation")
		MutateAndRestoreNetworkPolicy(t, ctx, kubeClient,
			pkg.TargetNamespace, allowMigratorPolicyName, reconcileTimeout)

		g.By("Mutating default-deny and waiting for reconciliation")
		MutateAndRestoreNetworkPolicy(t, ctx, kubeClient,
			pkg.TargetNamespace, defaultDenyPolicyName, reconcileTimeout)

		g.By("Checking NetworkPolicy-related events")
		LogNetworkPolicyEvents(t, ctx, kubeClient,
			[]string{pkg.OperatorNamespace, pkg.TargetNamespace}, allowMigratorPolicyName)
		LogNetworkPolicyEvents(t, ctx, kubeClient,
			[]string{pkg.OperatorNamespace, pkg.TargetNamespace}, defaultDenyPolicyName)
	})

	// =====================================================================
	// Operator namespace (openshift-kube-storage-version-migrator-operator)
	// Policies: allow-operator, default-deny (CVO-managed via manifests/)
	// =====================================================================

	g.It("[Operator][NetworkPolicy] should ensure NetworkPolicies are defined in the operator namespace", func() {
		t := g.GinkgoTB()
		ctx := context.Background()
		kubeClient := newKubeClient()

		policies, err := kubeClient.NetworkingV1().NetworkPolicies(pkg.OperatorNamespace).List(ctx, metav1.ListOptions{})
		o.Expect(err).NotTo(o.HaveOccurred())
		LogPolicyNames(t, pkg.OperatorNamespace, policies.Items)

		g.By("Checking default-deny policy exists")
		o.Expect(HasDefaultDeny(policies.Items)).To(o.BeTrue(),
			"expected a default-deny policy in %s", pkg.OperatorNamespace)

		g.By("Checking allow-operator policy exists")
		found := false
		for _, p := range policies.Items {
			if p.Name == allowOperatorPolicyName {
				found = true
				break
			}
		}
		o.Expect(found).To(o.BeTrue(),
			"expected %s policy in %s", allowOperatorPolicyName, pkg.OperatorNamespace)
	})

	g.It("[Operator][NetworkPolicy] should validate allow-operator policy structure", func() {
		t := g.GinkgoTB()
		ctx := context.Background()
		kubeClient := newKubeClient()

		policy := GetNetworkPolicy(t, ctx, kubeClient, pkg.OperatorNamespace, allowOperatorPolicyName)
		LogNetworkPolicySummary(t, allowOperatorPolicyName, policy)
		LogNetworkPolicyDetails(t, allowOperatorPolicyName, policy)

		g.By("Validating pod selector targets operator pods")
		RequirePodSelectorLabel(t, policy, "app", "kube-storage-version-migrator-operator")

		g.By("Validating ingress on port 8443 (metrics)")
		o.Expect(policy.Spec.Ingress).NotTo(o.BeEmpty(), "should have ingress rules")
		RequireIngressPort(t, policy, corev1.ProtocolTCP, 8443)

		g.By("Validating ingress allows from all sources (metrics scraping)")
		RequireIngressAllowAll(t, policy, 8443)

		g.By("Validating unrestricted egress")
		RequireUnrestrictedEgress(t, policy)

		g.By("Validating policy types include both Ingress and Egress")
		o.Expect(policy.Spec.PolicyTypes).To(o.ContainElement(networkingv1.PolicyTypeIngress))
		o.Expect(policy.Spec.PolicyTypes).To(o.ContainElement(networkingv1.PolicyTypeEgress))
	})

	g.It("[Operator][NetworkPolicy] should validate default-deny policy in operator namespace", func() {
		t := g.GinkgoTB()
		ctx := context.Background()
		kubeClient := newKubeClient()

		policy := GetNetworkPolicy(t, ctx, kubeClient, pkg.OperatorNamespace, defaultDenyPolicyName)
		LogNetworkPolicySummary(t, defaultDenyPolicyName, policy)

		g.By("Validating pod selector is empty (applies to all pods)")
		RequireEmptyPodSelector(t, policy)

		g.By("Validating policy types include both Ingress and Egress")
		o.Expect(policy.Spec.PolicyTypes).To(o.ContainElement(networkingv1.PolicyTypeIngress))
		o.Expect(policy.Spec.PolicyTypes).To(o.ContainElement(networkingv1.PolicyTypeEgress))

		g.By("Validating no allow rules are present")
		o.Expect(policy.Spec.Ingress).To(o.BeEmpty(), "default-deny should have no ingress rules")
		o.Expect(policy.Spec.Egress).To(o.BeEmpty(), "default-deny should have no egress rules")
	})

	g.It("[Operator][NetworkPolicy] should verify operator egress connectivity", func() {
		t := g.GinkgoTB()
		ctx := context.Background()
		kubeClient := newKubeClient()

		operatorLabels := map[string]string{"app": "kube-storage-version-migrator-operator"}

		g.By("Verifying egress to API server on port 443")
		kubeSvc, err := kubeClient.CoreV1().Services("default").Get(ctx, "kubernetes", metav1.GetOptions{})
		o.Expect(err).NotTo(o.HaveOccurred())
		ExpectConnectivity(ctx, t, kubeClient, pkg.OperatorNamespace, operatorLabels, []string{kubeSvc.Spec.ClusterIP}, 443, true)

		g.By("Verifying egress to DNS on port 53")
		dnsSvc, err := kubeClient.CoreV1().Services("openshift-dns").Get(ctx, "dns-default", metav1.GetOptions{})
		o.Expect(err).NotTo(o.HaveOccurred())
		ExpectConnectivity(ctx, t, kubeClient, pkg.OperatorNamespace, operatorLabels, []string{dnsSvc.Spec.ClusterIP}, 53, true)
	})

	g.It("[Operator][NetworkPolicy] should allow ingress to operator metrics endpoint", func() {
		t := g.GinkgoTB()
		ctx := context.Background()
		kubeClient := newKubeClient()

		pods, err := kubeClient.CoreV1().Pods(pkg.OperatorNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=kube-storage-version-migrator-operator",
		})
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(pods.Items).NotTo(o.BeEmpty(), "operator pod should exist")
		operatorPodIP := pods.Items[0].Status.PodIP

		g.By("Testing ingress from any namespace to metrics port 8443")
		testLabels := map[string]string{"test": "ingress-test"}
		ExpectConnectivity(ctx, t, kubeClient, "default", testLabels, []string{operatorPodIP}, 8443, true)
	})
})
