package deploymentcontroller

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	operatorv1 "github.com/openshift/api/operator/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func Test_setOperandLogLevel(t *testing.T) {

	testCases := []struct {
		logLevel operatorv1.LogLevel
		command  []string
		expected []string
	}{
		{
			logLevel: operatorv1.Debug,
			command:  []string{"arg0", "arg1", "arg2"},
			expected: []string{"arg0", "arg1", "arg2", "--v=4"},
		},
		{
			logLevel: operatorv1.Trace,
			command:  []string{"arg0", "--v=2", "arg2"},
			expected: []string{"arg0", "--v=6", "arg2"},
		},
		{
			logLevel: operatorv1.TraceAll,
			command:  []string{"arg0", "arg1", "--v=2"},
			expected: []string{"arg0", "arg1", "--v=8"},
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			spec := &operatorv1.OperatorSpec{
				LogLevel: tc.logLevel,
			}
			deployment := &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:    "migrator",
						Command: tc.command,
					}},
				}}},
			}
			_ = setOperandLogLevel(spec, deployment)
			if !cmp.Equal(deployment.Spec.Template.Spec.Containers[0].Command, tc.expected) {
				t.Fatal(cmp.Diff(deployment.Spec.Template.Spec.Containers[0].Command, tc.expected))
			}
		})
	}

	t.Run("", func(t *testing.T) {
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
