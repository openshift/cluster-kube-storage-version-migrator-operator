package e2e

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const (
	DefaultAgnhostImage = "registry.k8s.io/e2e-test-images/agnhost:2.45"
)

// ----- IP helpers -----

func IsIPv6(ip string) bool {
	return net.ParseIP(ip) != nil && strings.Contains(ip, ":")
}

func FormatIPPort(ip string, port int32) string {
	if IsIPv6(ip) {
		return fmt.Sprintf("[%s]:%d", ip, port)
	}
	return fmt.Sprintf("%s:%d", ip, port)
}

// PodIPs returns all IP addresses assigned to a pod (dual-stack aware).
func PodIPs(pod *corev1.Pod) []string {
	var ips []string
	for _, podIP := range pod.Status.PodIPs {
		if podIP.IP != "" {
			ips = append(ips, podIP.IP)
		}
	}
	if len(ips) == 0 && pod.Status.PodIP != "" {
		ips = append(ips, pod.Status.PodIP)
	}
	return ips
}

// ----- Pod / namespace construction -----

func netexecPod(name, namespace string, labels map[string]string, port int32, image string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot:   boolptr(true),
				RunAsUser:      int64ptr(1001),
				SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
			},
			Containers: []corev1.Container{
				{
					Name:  "netexec",
					Image: image,
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: boolptr(false),
						Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
						RunAsNonRoot:             boolptr(true),
						RunAsUser:                int64ptr(1001),
					},
					Command: []string{"/agnhost"},
					Args:    []string{"netexec", fmt.Sprintf("--http-port=%d", port)},
					Ports: []corev1.ContainerPort{
						{ContainerPort: port},
					},
				},
			},
		},
	}
}

// CreateTestNamespace creates a unique namespace for the test and returns its
// name along with a cleanup function that deletes it.
func CreateTestNamespace(ctx context.Context, t testing.TB, kubeClient kubernetes.Interface, prefix string) (string, func()) {
	t.Helper()
	name := fmt.Sprintf("%s-%s", prefix, rand.String(5))
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	_, err := kubeClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create test namespace %s: %v", name, err)
	}
	t.Logf("created test namespace %s", name)
	return name, func() {
		if delErr := kubeClient.CoreV1().Namespaces().Delete(context.Background(), name, metav1.DeleteOptions{}); delErr != nil {
			t.Logf("warning: failed to delete test namespace %s: %v", name, delErr)
		}
	}
}

// CreateServerPod creates an agnhost netexec server pod in the given namespace,
// waits for it to be Ready, and returns all its PodIPs along with a cleanup
// function.
func CreateServerPod(ctx context.Context, t testing.TB, kubeClient kubernetes.Interface, namespace, name string, labels map[string]string, port int32) ([]string, func()) {
	t.Helper()
	t.Logf("creating server pod %s/%s port=%d labels=%v", namespace, name, port, labels)

	pod := netexecPod(name, namespace, labels, port, DefaultAgnhostImage)
	_, err := kubeClient.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create server pod %s/%s: %v", namespace, name, err)
	}

	if err := waitForPodReady(ctx, kubeClient, namespace, name); err != nil {
		t.Fatalf("server pod %s/%s never became ready: %v", namespace, name, err)
	}

	created, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get server pod %s/%s: %v", namespace, name, err)
	}

	ips := PodIPs(created)
	if len(ips) == 0 {
		t.Fatalf("server pod %s/%s has no IPs", namespace, name)
	}
	t.Logf("server pod %s/%s ips=%v", namespace, name, ips)

	return ips, func() {
		t.Logf("deleting server pod %s/%s", namespace, name)
		_ = kubeClient.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	}
}

// ----- Connectivity checks -----

// RunConnectivityCheck creates an ephemeral agnhost connect pod in the given
// namespace with the specified labels, attempts a TCP connection to
// serverIP:port, and returns whether the connection succeeded.
func RunConnectivityCheck(ctx context.Context, kubeClient kubernetes.Interface, namespace string, labels map[string]string, serverIP string, port int32) (bool, error) {
	name := fmt.Sprintf("np-client-%s", rand.String(5))

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot:   boolptr(true),
				RunAsUser:      int64ptr(1001),
				SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
			},
			Containers: []corev1.Container{
				{
					Name:  "connect",
					Image: DefaultAgnhostImage,
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: boolptr(false),
						Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
						RunAsNonRoot:             boolptr(true),
						RunAsUser:                int64ptr(1001),
					},
					Command: []string{"/agnhost"},
					Args: []string{
						"connect",
						"--protocol=tcp",
						"--timeout=5s",
						FormatIPPort(serverIP, port),
					},
				},
			},
		},
	}

	_, err := kubeClient.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	defer func() {
		_ = kubeClient.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	}()

	if err := waitForPodCompletion(ctx, kubeClient, namespace, name); err != nil {
		return false, err
	}
	completed, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	if len(completed.Status.ContainerStatuses) == 0 {
		return false, fmt.Errorf("no container status recorded for pod %s", name)
	}
	terminated := completed.Status.ContainerStatuses[0].State.Terminated
	if terminated == nil {
		return false, fmt.Errorf("container in pod %s has not terminated", name)
	}
	return terminated.ExitCode == 0, nil
}

// ExpectConnectivity checks connectivity from a pod in the given namespace
// (with clientLabels) to each serverIP on the specified port. The check is
// retried for up to 2 minutes per IP. If the result does not match
// shouldSucceed the test is failed via t.Fatalf.
func ExpectConnectivity(ctx context.Context, t testing.TB, kubeClient kubernetes.Interface, namespace string, clientLabels map[string]string, serverIPs []string, port int32, shouldSucceed bool) {
	t.Helper()
	for _, ip := range serverIPs {
		family := "IPv4"
		if IsIPv6(ip) {
			family = "IPv6"
		}
		t.Logf("checking %s connectivity %s -> %s expected=%t", family, namespace, FormatIPPort(ip, port), shouldSucceed)
		if err := pollConnectivity(ctx, kubeClient, namespace, clientLabels, ip, port, shouldSucceed, 2*time.Minute); err != nil {
			t.Fatalf("connectivity check failed for %s %s -> %s (expected %t): %v", family, namespace, FormatIPPort(ip, port), shouldSucceed, err)
		}
	}
}

func pollConnectivity(ctx context.Context, kubeClient kubernetes.Interface, namespace string, clientLabels map[string]string, serverIP string, port int32, shouldSucceed bool, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, 5*time.Second, timeout, true, func(pollCtx context.Context) (bool, error) {
		succeeded, err := RunConnectivityCheck(pollCtx, kubeClient, namespace, clientLabels, serverIP, port)
		if err != nil {
			return false, err
		}
		return succeeded == shouldSucceed, nil
	})
}

// ----- Policy inspection helpers -----

func hasPort(ports []networkingv1.NetworkPolicyPort, protocol corev1.Protocol, port int32) bool {
	for _, p := range ports {
		if p.Protocol != nil && *p.Protocol != protocol {
			continue
		}
		if p.Port == nil || p.Port.IntValue() == int(port) {
			return true
		}
	}
	return false
}

func hasPortInIngress(rules []networkingv1.NetworkPolicyIngressRule, protocol corev1.Protocol, port int32) bool {
	for _, rule := range rules {
		if hasPort(rule.Ports, protocol, port) {
			return true
		}
	}
	return false
}

// HasDefaultDeny returns true if any policy in the list is a default-deny-all
// (empty podSelector with both Ingress and Egress policyTypes).
func HasDefaultDeny(policies []networkingv1.NetworkPolicy) bool {
	for _, policy := range policies {
		if len(policy.Spec.PodSelector.MatchLabels) != 0 || len(policy.Spec.PodSelector.MatchExpressions) != 0 {
			continue
		}
		hasIngress := false
		hasEgress := false
		for _, pt := range policy.Spec.PolicyTypes {
			if pt == networkingv1.PolicyTypeIngress {
				hasIngress = true
			}
			if pt == networkingv1.PolicyTypeEgress {
				hasEgress = true
			}
		}
		if hasIngress && hasEgress {
			return true
		}
	}
	return false
}

func hasIngressAllowAll(rules []networkingv1.NetworkPolicyIngressRule, port int32) bool {
	for _, rule := range rules {
		if !hasPort(rule.Ports, corev1.ProtocolTCP, port) {
			continue
		}
		if len(rule.From) == 0 {
			return true
		}
	}
	return false
}

// ----- Policy assertion helpers -----

// GetNetworkPolicy fetches a NetworkPolicy by namespace and name, failing the
// test if it does not exist.
func GetNetworkPolicy(t testing.TB, ctx context.Context, client kubernetes.Interface, namespace, name string) *networkingv1.NetworkPolicy {
	t.Helper()
	policy, err := client.NetworkingV1().NetworkPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get NetworkPolicy %s/%s: %v", namespace, name, err)
	}
	return policy
}

// RequirePodSelectorLabel asserts that the policy's podSelector contains the
// given key=value label.
func RequirePodSelectorLabel(t testing.TB, policy *networkingv1.NetworkPolicy, key, value string) {
	t.Helper()
	actual, ok := policy.Spec.PodSelector.MatchLabels[key]
	if !ok || actual != value {
		t.Fatalf("%s/%s: expected podSelector %s=%s, got %v", policy.Namespace, policy.Name, key, value, policy.Spec.PodSelector.MatchLabels)
	}
}

// RequireEmptyPodSelector asserts that the policy's podSelector is empty
// (selects all pods in the namespace).
func RequireEmptyPodSelector(t testing.TB, policy *networkingv1.NetworkPolicy) {
	t.Helper()
	if len(policy.Spec.PodSelector.MatchLabels) != 0 || len(policy.Spec.PodSelector.MatchExpressions) != 0 {
		t.Fatalf("%s/%s: expected empty podSelector, got matchLabels=%v matchExpressions=%v",
			policy.Namespace, policy.Name, policy.Spec.PodSelector.MatchLabels, policy.Spec.PodSelector.MatchExpressions)
	}
}

// RequireIngressPort asserts that the policy has an ingress rule with the
// specified protocol and port.
func RequireIngressPort(t testing.TB, policy *networkingv1.NetworkPolicy, protocol corev1.Protocol, port int32) {
	t.Helper()
	if !hasPortInIngress(policy.Spec.Ingress, protocol, port) {
		t.Fatalf("%s/%s: expected ingress port %s/%d", policy.Namespace, policy.Name, protocol, port)
	}
}

// RequireUnrestrictedEgress asserts that the policy has exactly one egress rule
// with no port and no destination restrictions (allows all egress).
func RequireUnrestrictedEgress(t testing.TB, policy *networkingv1.NetworkPolicy) {
	t.Helper()
	if len(policy.Spec.Egress) != 1 {
		t.Fatalf("%s/%s: expected exactly one egress rule for unrestricted egress, got %d rules",
			policy.Namespace, policy.Name, len(policy.Spec.Egress))
	}
	egressRule := policy.Spec.Egress[0]
	if len(egressRule.Ports) != 0 || len(egressRule.To) != 0 {
		t.Fatalf("%s/%s: expected unrestricted egress rule (no ports, no to), got ports=%v to=%v",
			policy.Namespace, policy.Name, egressRule.Ports, egressRule.To)
	}
}

// RequireIngressAllowAll asserts that the policy allows ingress from any source
// on the specified port.
func RequireIngressAllowAll(t testing.TB, policy *networkingv1.NetworkPolicy, port int32) {
	t.Helper()
	if !hasIngressAllowAll(policy.Spec.Ingress, port) {
		t.Fatalf("%s/%s: expected ingress allow-all on port %d", policy.Namespace, policy.Name, port)
	}
}

// ----- Policy reconciliation helpers -----

// RestoreNetworkPolicy deletes the given network policy and waits for the
// operator to recreate it with the expected spec. The timeout controls how long
// to wait for restoration.
func RestoreNetworkPolicy(t testing.TB, ctx context.Context, client kubernetes.Interface, expected *networkingv1.NetworkPolicy, timeout time.Duration) {
	t.Helper()
	namespace := expected.Namespace
	name := expected.Name
	t.Logf("deleting NetworkPolicy %s/%s and waiting for restoration", namespace, name)
	if err := client.NetworkingV1().NetworkPolicies(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		t.Fatalf("failed to delete NetworkPolicy %s/%s: %v", namespace, name, err)
	}
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		current, getErr := client.NetworkingV1().NetworkPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			if errors.IsNotFound(getErr) {
				return false, nil
			}
			return false, getErr
		}
		return equality.Semantic.DeepEqual(expected.Spec, current.Spec), nil
	})
	if err != nil {
		t.Fatalf("timed out waiting for NetworkPolicy %s/%s spec to be restored", namespace, name)
	}
	t.Logf("NetworkPolicy %s/%s spec restored after delete", namespace, name)
}

// MutateAndRestoreNetworkPolicy patches the policy's podSelector with a
// spurious label, then waits for the operator to reconcile it back to the
// original spec. The timeout controls how long to wait for reconciliation.
func MutateAndRestoreNetworkPolicy(t testing.TB, ctx context.Context, client kubernetes.Interface, namespace, name string, timeout time.Duration) {
	t.Helper()
	original := GetNetworkPolicy(t, ctx, client, namespace, name)
	t.Logf("mutating NetworkPolicy %s/%s (podSelector override) and waiting for reconciliation", namespace, name)
	patch := []byte(`{"spec":{"podSelector":{"matchLabels":{"np-reconcile":"mutated"}}}}`)
	_, err := client.NetworkingV1().NetworkPolicies(namespace).Patch(ctx, name, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		t.Fatalf("failed to patch NetworkPolicy %s/%s: %v", namespace, name, err)
	}

	err = wait.PollUntilContextTimeout(ctx, 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		current, getErr := client.NetworkingV1().NetworkPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			return false, getErr
		}
		return equality.Semantic.DeepEqual(original.Spec, current.Spec), nil
	})
	if err != nil {
		t.Fatalf("timed out waiting for NetworkPolicy %s/%s spec to be restored after mutation", namespace, name)
	}
	t.Logf("NetworkPolicy %s/%s spec restored after mutation", namespace, name)
}

// ----- Logging helpers -----

// LogPolicyNames logs the names of all NetworkPolicies in the given list.
func LogPolicyNames(t testing.TB, namespace string, policies []networkingv1.NetworkPolicy) {
	t.Helper()
	names := make([]string, 0, len(policies))
	for _, policy := range policies {
		names = append(names, policy.Name)
	}
	t.Logf("networkpolicies in %s: %v", namespace, names)
}

// LogNetworkPolicySummary logs a one-line summary of a NetworkPolicy.
func LogNetworkPolicySummary(t testing.TB, label string, policy *networkingv1.NetworkPolicy) {
	t.Helper()
	t.Logf("networkpolicy %s namespace=%s name=%s podSelector=%v policyTypes=%v ingress=%d egress=%d",
		label,
		policy.Namespace,
		policy.Name,
		policy.Spec.PodSelector.MatchLabels,
		policy.Spec.PolicyTypes,
		len(policy.Spec.Ingress),
		len(policy.Spec.Egress),
	)
}

// LogNetworkPolicyDetails logs detailed ingress and egress rules.
func LogNetworkPolicyDetails(t testing.TB, label string, policy *networkingv1.NetworkPolicy) {
	t.Helper()
	t.Logf("networkpolicy %s details:", label)
	t.Logf("  podSelector=%v policyTypes=%v", policy.Spec.PodSelector.MatchLabels, policy.Spec.PolicyTypes)
	for i, rule := range policy.Spec.Ingress {
		t.Logf("  ingress[%d]: ports=%s from=%s", i, formatPorts(rule.Ports), formatPeers(rule.From))
	}
	for i, rule := range policy.Spec.Egress {
		t.Logf("  egress[%d]: ports=%s to=%s", i, formatPorts(rule.Ports), formatPeers(rule.To))
	}
}

// LogNetworkPolicyEvents searches for NetworkPolicy-related events in the
// given namespaces (best-effort, does not fail).
//
// Events emitted by the resourceapply package in library-go use the operator
// Deployment as the InvolvedObject (not the NetworkPolicy itself).  The event
// Reason is prefixed with "NetworkPolicy" (e.g. NetworkPolicyCreated,
// NetworkPolicyUpdated, NetworkPolicyDeleted) and the event Message contains
// the full resource reference including the policy name.  Therefore this
// function matches events by:
//   - Reason starting with "NetworkPolicy", OR
//   - Message containing the policyName, OR
//   - InvolvedObject.Kind == "NetworkPolicy" (for any recorder that does
//     reference the policy directly).
//
// Callers should include the **operator** namespace in the namespaces list
// because that is where resourceapply records the events.
func LogNetworkPolicyEvents(t testing.TB, ctx context.Context, client kubernetes.Interface, namespaces []string, policyName string) {
	t.Helper()
	found := false
	_ = wait.PollUntilContextTimeout(ctx, 5*time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
		for _, namespace := range namespaces {
			eventList, err := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Logf("unable to list events in %s: %v", namespace, err)
				continue
			}
			for _, event := range eventList.Items {
				isNPEvent := false

				if strings.HasPrefix(event.Reason, "NetworkPolicy") {
					isNPEvent = true
				}
				if event.InvolvedObject.Kind == "NetworkPolicy" {
					isNPEvent = true
				}
				if policyName != "" && strings.Contains(event.Message, policyName) {
					isNPEvent = true
				}

				if isNPEvent {
					t.Logf("event in %s: type=%s reason=%s involvedObject=%s/%s message=%q",
						namespace, event.Type, event.Reason,
						event.InvolvedObject.Kind, event.InvolvedObject.Name,
						event.Message)
					found = true
				}
			}
		}
		if found {
			return true, nil
		}
		t.Logf("no NetworkPolicy events yet for %s (namespaces: %v)", policyName, namespaces)
		return false, nil
	})
	if !found {
		t.Logf("no NetworkPolicy events observed for %s (best-effort)", policyName)
	}
}

// ----- Format helpers -----

func formatPorts(ports []networkingv1.NetworkPolicyPort) string {
	if len(ports) == 0 {
		return "[]"
	}
	out := make([]string, 0, len(ports))
	for _, p := range ports {
		proto := "TCP"
		if p.Protocol != nil {
			proto = string(*p.Protocol)
		}
		if p.Port == nil {
			out = append(out, fmt.Sprintf("%s:any", proto))
			continue
		}
		out = append(out, fmt.Sprintf("%s:%s", proto, p.Port.String()))
	}
	return fmt.Sprintf("[%s]", strings.Join(out, ", "))
}

func formatPeers(peers []networkingv1.NetworkPolicyPeer) string {
	if len(peers) == 0 {
		return "[]"
	}
	out := make([]string, 0, len(peers))
	for _, peer := range peers {
		ns := formatSelector(peer.NamespaceSelector)
		pod := formatSelector(peer.PodSelector)
		if ns == "" && pod == "" {
			out = append(out, "{}")
			continue
		}
		out = append(out, fmt.Sprintf("ns=%s pod=%s", ns, pod))
	}
	return fmt.Sprintf("[%s]", strings.Join(out, ", "))
}

func formatSelector(sel *metav1.LabelSelector) string {
	if sel == nil {
		return ""
	}
	if len(sel.MatchLabels) == 0 && len(sel.MatchExpressions) == 0 {
		return "{}"
	}
	return fmt.Sprintf("labels=%v exprs=%v", sel.MatchLabels, sel.MatchExpressions)
}

// ----- Wait helpers -----

func waitForPodReady(ctx context.Context, kubeClient kubernetes.Interface, namespace, name string) error {
	return wait.PollUntilContextTimeout(ctx, 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if pod.Status.Phase != corev1.PodRunning {
			return false, nil
		}
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
}

func waitForPodCompletion(ctx context.Context, kubeClient kubernetes.Interface, namespace, name string) error {
	return wait.PollUntilContextTimeout(ctx, 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed, nil
	})
}

// ----- Utility helpers -----

func boolptr(value bool) *bool {
	return &value
}

func int64ptr(value int64) *int64 {
	return &value
}
