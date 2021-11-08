package deploymentcontroller

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/cluster-kube-storage-version-migrator-operator/bindata"
	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/deploymentcontroller"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openshift/cluster-kube-storage-version-migrator-operator/pkg"
)

func NewMigratorDeploymentController(
	kubeClient kubernetes.Interface,
	operatorClient v1helpers.OperatorClientWithFinalizers,
	kubeInformersForNamespaces v1helpers.KubeInformersForNamespaces,
	recorder events.Recorder,
) factory.Controller {
	return deploymentcontroller.NewDeploymentController(
		"KubeStorageVersionMigrator",
		bindata.MustAsset("kube-storage-version-migrator/deployment.yaml"),
		recorder,
		operatorClient,
		kubeClient,
		kubeInformersForNamespaces.InformersFor(pkg.TargetNamespace).Apps().V1().Deployments(),
		[]factory.Informer{
			kubeInformersForNamespaces.InformersFor(pkg.TargetNamespace).Core().V1().Secrets().Informer(),
		},
		[]deploymentcontroller.ManifestHookFunc{
			replaceAll("${IMAGE}", os.Getenv("IMAGE")),
		},
		setOperandLogLevel,
	)
}

func replaceAll(old, new string) deploymentcontroller.ManifestHookFunc {
	return func(spec *operatorv1.OperatorSpec, manifest []byte) ([]byte, error) {
		return bytes.ReplaceAll(manifest, []byte(old), []byte(new)), nil
	}
}

func setOperandLogLevel(spec *operatorv1.OperatorSpec, deployment *appsv1.Deployment) error {
	var i int
	var ok bool
	var c corev1.Container
	for i, c = range deployment.Spec.Template.Spec.Containers {
		ok = c.Name == "migrator"
		if ok {
			break
		}
	}
	if !ok {
		return fmt.Errorf("deployment does not contain a container named migrator")
	}
	v := 2
	switch spec.LogLevel {
	case operatorv1.TraceAll:
		v = 8
	case operatorv1.Trace:
		v = 6
	case operatorv1.Debug:
		v = 4
	}
	logLevelArg := fmt.Sprintf("--v=%d", v)

	container := &deployment.Spec.Template.Spec.Containers[i]
	command := container.Command[:0]
	var logLevelSet bool
	for _, arg := range container.Command {
		if strings.HasPrefix(arg, "--v=") {
			// replace existing log level arg
			command = append(command, logLevelArg)
			logLevelSet = true
			continue
		}
		command = append(command, arg)
	}
	if !logLevelSet {
		// append new log level arg
		command = append(command, logLevelArg)
		container.Command = command
	}
	return nil
}
