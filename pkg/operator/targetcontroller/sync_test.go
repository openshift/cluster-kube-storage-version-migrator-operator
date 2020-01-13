package targetcontroller

import (
	v12 "github.com/openshift/api/operator/v1"
	v1 "k8s.io/api/apps/v1"
	v13 "k8s.io/api/core/v1"
	v14 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"testing"
)

func TestResolveImageReference(t *testing.T) {
	controller := TargetController{
		imagePullSpec:         "image-pull:spec",
		operatorImagePullSpec: "operator-image-pull:spec",
	}
	deployment := v1.Deployment{
		Spec: v1.DeploymentSpec{
			Template: v13.PodTemplateSpec{
				Spec: v13.PodSpec{
					InitContainers: []v13.Container{
						{
							Image: "${IMAGE}",
						},
					},
					Containers: []v13.Container{
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
	deployment := &v1.Deployment{
		Status: v1.DeploymentStatus{
			AvailableReplicas: 1,
		},
	}
	status := &v12.KubeStorageVersionMigratorStatus{}
	manageOperatorStatusAvailable(deployment, status)
	t.Log(mergepatch.ToYAMLOrError(status))
}

func TestManageOperatorStatusProgressing(t *testing.T) {
	deployment := &v1.Deployment{
		ObjectMeta: v14.ObjectMeta{
			Generation: 1,
		},
		Status: v1.DeploymentStatus{
			ObservedGeneration: 1,
		},
	}
	status := &v12.KubeStorageVersionMigratorStatus{
		OperatorStatus: v12.OperatorStatus{
			ObservedGeneration: 1,
		},
	}
	manageOperatorStatusProgressing(deployment, status, 1)
	t.Log(mergepatch.ToYAMLOrError(status))
}
