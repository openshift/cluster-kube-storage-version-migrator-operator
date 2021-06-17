package operator

import (
	"context"
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	configv1client "github.com/openshift/client-go/config/clientset/versioned"
	configinformers "github.com/openshift/client-go/config/informers/externalversions"
	operatorv1client "github.com/openshift/client-go/operator/clientset/versioned/typed/operator/v1"
	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/openshift/library-go/pkg/operator/genericoperatorclient"
	"github.com/openshift/library-go/pkg/operator/loglevel"
	"github.com/openshift/library-go/pkg/operator/status"

	"github.com/openshift/cluster-kube-storage-version-migrator-operator/pkg/operator/targetcontroller"
)

const (
	OperatorNamespace = "openshift-kube-storage-version-migrator-operator"
	TargetNamespace   = "openshift-kube-storage-version-migrator"
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

	operatorConfigClient, err := operatorv1client.NewForConfig(cc.KubeConfig)
	if err != nil {
		return err
	}

	genericOperatorConfigClient, dynamicInformers, err := genericoperatorclient.NewClusterScopedOperatorClient(
		cc.KubeConfig, operatorv1.GroupVersion.WithResource("kubestorageversionmigrators"))
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

	kubeInformersForTargetNamespace := informers.NewSharedInformerFactoryWithOptions(kubeClient, 10*time.Minute,
		informers.WithNamespace(TargetNamespace),
	)
	targetController := targetcontroller.NewTargetController(
		kubeClient,
		genericOperatorConfigClient,
		operatorConfigClient.KubeStorageVersionMigrators(),
		kubeInformersForTargetNamespace.Core().V1().Secrets(),
		kubeInformersForTargetNamespace.Apps().V1().Deployments(),
		os.Getenv("IMAGE"),
		os.Getenv("OPERATOR_IMAGE"),
		cc.EventRecorder,
		versionRecorder,
	)

	configInformers := configinformers.NewSharedInformerFactory(configClient, 10*time.Minute)

	statusController := status.NewClusterOperatorStatusController(
		"kube-storage-version-migrator",
		[]configv1.ObjectReference{
			{Group: "operator.openshift.io", Resource: "kubestorageversionmigrators", Name: "cluster"},
			{Group: "migration.k8s.io", Resource: "storageversionmigrations"},
			{Resource: "namespaces", Name: TargetNamespace},
			{Resource: "namespaces", Name: OperatorNamespace},
		},
		configClient.ConfigV1(),
		configInformers.Config().V1().ClusterOperators(),
		genericOperatorConfigClient,
		versionRecorder,
		cc.EventRecorder,
	)

	loggingController := loglevel.NewClusterOperatorLoggingController(genericOperatorConfigClient, cc.EventRecorder)

	configInformers.Start(ctx.Done())
	dynamicInformers.Start(ctx.Done())
	kubeInformersForTargetNamespace.Start(ctx.Done())

	go statusController.Run(ctx, 1)
	go targetController.Run(1, ctx.Done())
	go loggingController.Run(ctx, 1)

	<-ctx.Done()
	return fmt.Errorf("stopped")
}
