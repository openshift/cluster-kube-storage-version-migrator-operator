package deploymentcontroller

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	configv1listers "github.com/openshift/client-go/config/listers/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

func Test_setOperandLogLevel(t *testing.T) {

	testCases := []struct {
		name     string
		logLevel operatorv1.LogLevel
		command  []string
		args     []string
		expected []string
	}{
		{
			name:     "CmdAppend",
			logLevel: operatorv1.Debug,
			command:  []string{"arg0", "arg1", "arg2"},
			expected: []string{"arg0", "arg1", "arg2", "--v=4"},
		},
		{
			name:     "CmdReplaceMid",
			logLevel: operatorv1.Trace,
			command:  []string{"arg0", "--v=2", "arg2"},
			expected: []string{"arg0", "--v=6", "arg2"},
		},
		{
			name:     "CmdReplaceEnd",
			logLevel: operatorv1.TraceAll,
			command:  []string{"arg0", "arg1", "--v=2"},
			expected: []string{"arg0", "arg1", "--v=8"},
		},
		{
			name:     "ArgsAppend",
			logLevel: operatorv1.Debug,
			command:  []string{"arg0"},
			args:     []string{"arg1", "arg2", "arg3"},
			expected: []string{"arg0", "arg1", "arg2", "arg3", "--v=4"},
		},
		{
			name:     "ArgsReplaceMid",
			logLevel: operatorv1.Trace,
			command:  []string{"arg0"},
			args:     []string{"arg1", "--v=2", "arg3"},
			expected: []string{"arg0", "arg1", "--v=6", "arg3"},
		},
		{
			name:     "ArgsReplaceEnd",
			logLevel: operatorv1.TraceAll,
			command:  []string{"arg0"},
			args:     []string{"arg1", "arg2", "--v=2"},
			expected: []string{"arg0", "arg1", "arg2", "--v=8"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			spec := &operatorv1.OperatorSpec{
				LogLevel: tc.logLevel,
			}
			deployment := &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:    "migrator",
						Command: tc.command,
						Args:    tc.args,
					}},
				}}},
			}
			_ = setOperandLogLevel(spec, deployment)
			c := deployment.Spec.Template.Spec.Containers[0]
			actual := append(append([]string{}, c.Command...), c.Args...)
			if !cmp.Equal(tc.expected, actual) {
				t.Fatal(cmp.Diff(tc.expected, actual))
			}
		})
	}

	t.Run("OtherContainer", func(t *testing.T) {
		spec := &operatorv1.OperatorSpec{LogLevel: operatorv1.Debug}
		deployment := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name: "not_migrator",
				}},
			}}},
		}
		if err := setOperandLogLevel(spec, deployment); err == nil {
			t.Fatal("expected error")
		}
	})

}

func fakeNodeLister(nodes ...*corev1.Node) corev1listers.NodeLister {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	for _, n := range nodes {
		_ = indexer.Add(n)
	}
	return corev1listers.NewNodeLister(indexer)
}

func fakeInfrastructureLister(infra *configv1.Infrastructure) configv1listers.InfrastructureLister {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	_ = indexer.Add(infra)
	return configv1listers.NewInfrastructureLister(indexer)
}

func newControlPlaneNode(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"node-role.kubernetes.io/control-plane": ""},
		},
	}
}

func newInfrastructure(topology configv1.TopologyMode) *configv1.Infrastructure {
	return &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Status: configv1.InfrastructureStatus{
			ControlPlaneTopology: topology,
		},
	}
}

func Test_setDesiredReplicas(t *testing.T) {
	testCases := []struct {
		name             string
		topology         configv1.TopologyMode
		nodes            []*corev1.Node
		expectedReplicas int32
	}{
		{
			name:     "HighlyAvailable with 3 control-plane nodes",
			topology: configv1.HighlyAvailableTopologyMode,
			nodes: []*corev1.Node{
				newControlPlaneNode("master-0"),
				newControlPlaneNode("master-1"),
				newControlPlaneNode("master-2"),
			},
			expectedReplicas: 3,
		},
		{
			name:             "SingleReplica with 1 control-plane node",
			topology:         configv1.SingleReplicaTopologyMode,
			nodes:            []*corev1.Node{newControlPlaneNode("master-0")},
			expectedReplicas: 1,
		},
		{
			name:             "External topology defaults to 2",
			topology:         configv1.ExternalTopologyMode,
			nodes:            nil,
			expectedReplicas: 2,
		},
		{
			name:     "External topology ignores visible nodes",
			topology: configv1.ExternalTopologyMode,
			nodes: []*corev1.Node{
				newControlPlaneNode("master-0"),
				newControlPlaneNode("master-1"),
				newControlPlaneNode("master-2"),
			},
			expectedReplicas: 2,
		},
		{
			name:             "HighlyAvailable with no nodes defaults to 1",
			topology:         configv1.HighlyAvailableTopologyMode,
			nodes:            nil,
			expectedReplicas: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nodeLister := fakeNodeLister(tc.nodes...)
			infraLister := fakeInfrastructureLister(newInfrastructure(tc.topology))

			hook := setDesiredReplicas(infraLister, nodeLister)

			deployment := &appsv1.Deployment{}
			if err := hook(&operatorv1.OperatorSpec{}, deployment); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if deployment.Spec.Replicas == nil {
				t.Fatal("expected replicas to be set")
			}
			if *deployment.Spec.Replicas != tc.expectedReplicas {
				t.Fatalf("expected %d replicas, got %d", tc.expectedReplicas, *deployment.Spec.Replicas)
			}
		})
	}
}

func Test_setControlPlaneNodeSelector(t *testing.T) {
	testCases := []struct {
		name             string
		topology         configv1.TopologyMode
		expectedSelector map[string]string
	}{
		{
			name:             "HighlyAvailable sets control-plane nodeSelector",
			topology:         configv1.HighlyAvailableTopologyMode,
			expectedSelector: map[string]string{"node-role.kubernetes.io/control-plane": ""},
		},
		{
			name:             "SingleReplica sets control-plane nodeSelector",
			topology:         configv1.SingleReplicaTopologyMode,
			expectedSelector: map[string]string{"node-role.kubernetes.io/control-plane": ""},
		},
		{
			name:             "External topology does not set nodeSelector",
			topology:         configv1.ExternalTopologyMode,
			expectedSelector: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			infraLister := fakeInfrastructureLister(newInfrastructure(tc.topology))
			hook := setControlPlaneNodeSelector(infraLister)

			deployment := &appsv1.Deployment{}
			if err := hook(&operatorv1.OperatorSpec{}, deployment); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.expectedSelector, deployment.Spec.Template.Spec.NodeSelector); diff != "" {
				t.Fatalf("unexpected nodeSelector (-want +got):\n%s", diff)
			}
		})
	}
}

// Verify fakeNodeLister only returns nodes matching the control-plane selector.
func Test_setDesiredReplicas_nodeFiltering(t *testing.T) {
	// The node lister in setDesiredReplicas uses a label selector, but our fake
	// lister returns all indexed nodes. This test documents that in production
	// the informer's list options filter nodes, and the lister's List(selector)
	// additionally filters. We test with the selector to ensure correctness.
	nodes := []*corev1.Node{
		newControlPlaneNode("master-0"),
		{ObjectMeta: metav1.ObjectMeta{Name: "worker-0", Labels: map[string]string{"node-role.kubernetes.io/worker": ""}}},
		newControlPlaneNode("master-1"),
		newControlPlaneNode("master-2"),
	}

	// Build a real indexer with all nodes to test the label selector in setDesiredReplicas
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	for _, n := range nodes {
		objs := []runtime.Object{n}
		for _, o := range objs {
			_ = indexer.Add(o)
		}
	}
	nodeLister := corev1listers.NewNodeLister(indexer)
	infraLister := fakeInfrastructureLister(newInfrastructure(configv1.HighlyAvailableTopologyMode))

	hook := setDesiredReplicas(infraLister, nodeLister)

	deployment := &appsv1.Deployment{}
	if err := hook(&operatorv1.OperatorSpec{}, deployment); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The label selector inside setDesiredReplicas filters to control-plane nodes only
	if *deployment.Spec.Replicas != 3 {
		t.Fatalf("expected 3 replicas (control-plane nodes only), got %d", *deployment.Spec.Replicas)
	}
}
