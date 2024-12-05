package operator

import (
	"context"
	"fmt"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	configv1client "github.com/openshift/client-go/config/clientset/versioned"
	configinformers "github.com/openshift/client-go/config/informers/externalversions"
	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/openshift/library-go/pkg/operator/genericoperatorclient"
	"github.com/openshift/library-go/pkg/operator/loglevel"
	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	"github.com/openshift/library-go/pkg/operator/staleconditions"
	"github.com/openshift/library-go/pkg/operator/staticresourcecontroller"
	"github.com/openshift/library-go/pkg/operator/status"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	applyoperatorv1 "github.com/openshift/client-go/operator/applyconfigurations/operator/v1"
	"github.com/openshift/cluster-kube-storage-version-migrator-operator/bindata"
	"github.com/openshift/cluster-kube-storage-version-migrator-operator/pkg"
	"github.com/openshift/cluster-kube-storage-version-migrator-operator/pkg/operator/deploymentcontroller"
	"github.com/openshift/cluster-kube-storage-version-migrator-operator/pkg/operator/staticconditionscontroller"
)

func RunOperator(ctx context.Context, cc *controllercmd.ControllerContext) error {

	kubeClient, err := kubernetes.NewForConfig(cc.ProtoKubeConfig)
	if err != nil {
		return err
	}

	configClient, err := configv1client.NewForConfig(cc.KubeConfig)
	if err != nil {
		return err
	}

	operatorClient, dynamicInformers, err := genericoperatorclient.NewClusterScopedOperatorClient(
		cc.Clock,
		cc.KubeConfig,
		operatorv1.GroupVersion.WithResource("kubestorageversionmigrators"),
		operatorv1.GroupVersion.WithKind("KubeStorageVersionMigrator"),
		extractOperatorSpec,
		extractOperatorStatus,
	)
	if err != nil {
		return err
	}

	clusterOperator, err := configClient.ConfigV1().ClusterOperators().Get(ctx, "kube-storage-version-migrator", metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	versionRecorder := status.NewVersionGetter()
	for _, version := range clusterOperator.Status.Versions {
		versionRecorder.SetVersion(version.Name, version.Version)
	}
	versionRecorder.SetVersion("operator", status.VersionForOperatorFromEnv())

	kubeInformersForNamespaces := v1helpers.NewKubeInformersForNamespaces(kubeClient, pkg.TargetNamespace)

	staticResourceController := staticresourcecontroller.NewStaticResourceController(
		"KubeStorageVersionMigratorStaticResources",
		bindata.Asset,
		[]string{
			"kube-storage-version-migrator/namespace.yaml",
			"kube-storage-version-migrator/serviceaccount.yaml",
			"kube-storage-version-migrator/roles.yaml",
		},
		(&resourceapply.ClientHolder{}).WithKubernetes(kubeClient),
		operatorClient,
		cc.EventRecorder,
	)

	migratorDeploymentController := deploymentcontroller.NewMigratorDeploymentController(
		kubeClient,
		operatorClient,
		kubeInformersForNamespaces,
		cc.EventRecorder,
	)

	configInformers := configinformers.NewSharedInformerFactory(configClient, 10*time.Minute)

	statusController := status.NewClusterOperatorStatusController(
		"kube-storage-version-migrator",
		[]configv1.ObjectReference{
			{Group: "operator.openshift.io", Resource: "kubestorageversionmigrators", Name: "cluster"},
			{Group: "migration.k8s.io", Resource: "storageversionmigrations"},
			{Resource: "namespaces", Name: pkg.TargetNamespace},
			{Resource: "namespaces", Name: pkg.OperatorNamespace},
		},
		configClient.ConfigV1(),
		configInformers.Config().V1().ClusterOperators(),
		operatorClient,
		versionRecorder,
		cc.EventRecorder,
	)

	staticConditionsController := staticconditionscontroller.NewStaticConditionsController(
		operatorClient, cc.EventRecorder,
		operatorv1.OperatorCondition{Type: "Default" + operatorv1.OperatorStatusTypeUpgradeable, Status: operatorv1.ConditionTrue, Reason: "Default"},
	)

	staleConditionsController := staleconditions.NewRemoveStaleConditionsController(
		"kube-storage-version-migrator",
		[]string{"Available", "Progressing", "TargetDegraded", "DefaultUpgradable"},
		operatorClient,
		cc.EventRecorder,
	)

	loggingController := loglevel.NewClusterOperatorLoggingController(operatorClient, cc.EventRecorder)

	configInformers.Start(ctx.Done())
	dynamicInformers.Start(ctx.Done())
	kubeInformersForNamespaces.Start(ctx.Done())

	go statusController.Run(ctx, 1)
	go staticResourceController.Run(ctx, 1)
	go migratorDeploymentController.Run(ctx, 1)
	go staticConditionsController.Run(ctx, 1)
	go staleConditionsController.Run(ctx, 1)
	go loggingController.Run(ctx, 1)

	<-ctx.Done()
	return nil
}

func extractOperatorSpec(obj *unstructured.Unstructured, fieldManager string) (*applyoperatorv1.OperatorSpecApplyConfiguration, error) {
	castObj := &operatorv1.KubeStorageVersionMigrator{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, castObj); err != nil {
		return nil, fmt.Errorf("unable to convert to KubeStorageVersionMigrator: %w", err)
	}
	ret, err := applyoperatorv1.ExtractKubeStorageVersionMigrator(castObj, fieldManager)
	if err != nil {
		return nil, fmt.Errorf("unable to extract fields for %q: %w", fieldManager, err)
	}
	if ret.Spec == nil {
		return nil, nil
	}
	return &ret.Spec.OperatorSpecApplyConfiguration, nil
}

func extractOperatorStatus(obj *unstructured.Unstructured, fieldManager string) (*applyoperatorv1.OperatorStatusApplyConfiguration, error) {
	castObj := &operatorv1.KubeStorageVersionMigrator{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, castObj); err != nil {
		return nil, fmt.Errorf("unable to convert to KubeStorageVersionMigrator: %w", err)
	}
	ret, err := applyoperatorv1.ExtractKubeStorageVersionMigratorStatus(castObj, fieldManager)
	if err != nil {
		return nil, fmt.Errorf("unable to extract fields for %q: %w", fieldManager, err)
	}

	if ret.Status == nil {
		return nil, nil
	}
	return &ret.Status.OperatorStatusApplyConfiguration, nil
}
