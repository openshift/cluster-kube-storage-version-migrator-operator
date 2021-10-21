package targetcontroller

import (
	"fmt"
	"strings"
	"testing"

	operatorv1 "github.com/openshift/api/operator/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/mergepatch"
)

func TestResolveImageReference(t *testing.T) {
	controller := TargetController{
		imagePullSpec:         "image-pull:spec",
		operatorImagePullSpec: "operator-image-pull:spec",
	}
	deployment := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Image: "${IMAGE}",
						},
					},
					Containers: []corev1.Container{
						{
							Image: "${IMAGE}",
						},
						{
							Image: "${OPERATOR_IMAGE}",
						},
					},
				},
			},
		},
	}
	spec := &deployment.Spec.Template.Spec
	spec.InitContainers, _ = controller.resolveImageReferences(spec.InitContainers)
	spec.Containers, _ = controller.resolveImageReferences(spec.Containers)
	t.Log(mergepatch.ToYAMLOrError(deployment))
}

func TestManageOperatorStatusAvailable(t *testing.T) {
	deployment := &appsv1.Deployment{
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
		},
	}
	status := &operatorv1.KubeStorageVersionMigratorStatus{}
	manageOperatorStatusAvailable(deployment, status)
	t.Log(mergepatch.ToYAMLOrError(status))
}

func TestManageOperatorStatusProgressing(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Generation: 1,
		},
		Status: appsv1.DeploymentStatus{
			ObservedGeneration: 1,
		},
	}
	status := &operatorv1.KubeStorageVersionMigratorStatus{
		OperatorStatus: operatorv1.OperatorStatus{
			ObservedGeneration: 1,
		},
	}
	manageOperatorStatusProgressing(deployment, nil, status, 1)
	t.Log(mergepatch.ToYAMLOrError(status))
}

func TestManageOperatorStatusUpgradeable(t *testing.T) {
	status := &operatorv1.KubeStorageVersionMigratorStatus{}
	manageOperatorStatusUpgradeable(status)
	if !strings.HasSuffix(status.Conditions[0].Type, operatorv1.OperatorStatusTypeUpgradeable) {
		t.Errorf("expecting an Upgradeable condition")
	}
	if status.Conditions[0].Status != operatorv1.ConditionTrue {
		t.Errorf("expecting condition status to be %q", operatorv1.ConditionTrue)
	}
	if t.Failed() {
		t.Log(mergepatch.ToYAMLOrError(status))
	}

}

func TestManageOperatorStatusProgressingSyncErr(t *testing.T) {
	var errors []error
	errors = append(errors, fmt.Errorf("syncErr"))
	var statusTrue operatorv1.ConditionStatus = "True"
	status := &operatorv1.KubeStorageVersionMigratorStatus{}
	manageOperatorStatusProgressing(nil, errors, status, 1)
	if status.OperatorStatus.Conditions[0].Status != statusTrue {
		t.Errorf("Expected Progressing %v, got %v", statusTrue, status.OperatorStatus.Conditions[0].Status)
	}
	t.Log(mergepatch.ToYAMLOrError(status))
}
