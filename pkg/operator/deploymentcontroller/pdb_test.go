package deploymentcontroller

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openshift/cluster-kube-storage-version-migrator-operator/bindata"
	appsv1 "k8s.io/api/apps/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"
)

func TestMigratorPDBSelector(t *testing.T) {
	load := func(name string, out interface{}) {
		t.Helper()
		data, err := bindata.Asset("kube-storage-version-migrator/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if err := yaml.UnmarshalStrict(data, out); err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}
	}

	var deployment appsv1.Deployment
	var pdb policyv1.PodDisruptionBudget
	load("deployment.yaml", &deployment)
	load("pdb.yaml", &pdb)

	selector, err := metav1.LabelSelectorAsSelector(pdb.Spec.Selector)
	if err != nil {
		t.Fatal(err)
	}

	const component = "kube-storage-version-migrator.openshift.io/component"
	cases := []struct {
		name      string
		podLabels map[string]string
		want      bool
	}{
		{
			name:      "real migrator",
			podLabels: deployment.Spec.Template.Labels,
			want:      true,
		},
		{
			name:      "e2e client",
			podLabels: map[string]string{"app": "migrator"},
			want:      false,
		},
		{
			name:      "wrong component",
			podLabels: map[string]string{"app": "migrator", component: "other"},
			want:      false,
		},
		{
			name:      "missing app",
			podLabels: map[string]string{component: "migrator"},
			want:      false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := selector.Matches(labels.Set(tc.podLabels)); got != tc.want {
				t.Errorf("PDB matches %v = %t, want %t", tc.podLabels, got, tc.want)
			}
		})
	}

	wantSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "migrator"},
	}
	if diff := cmp.Diff(wantSelector, deployment.Spec.Selector); diff != "" {
		t.Errorf("Deployment selector changed (-want +got):\n%s", diff)
	}
	if deployment.Spec.Template.Labels[component] != "migrator" {
		t.Error("Pod template is missing the migrator component label")
	}
	if pdb.Spec.MaxUnavailable == nil ||
		pdb.Spec.MaxUnavailable.Type != intstr.Int ||
		pdb.Spec.MaxUnavailable.IntVal != 1 ||
		pdb.Spec.MinAvailable != nil {
		t.Error("expected maxUnavailable: 1 and no minAvailable")
	}
}
