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
