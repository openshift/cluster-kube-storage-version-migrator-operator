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
	"github.com/openshift/library-go/pkg/operator/staticresourcecontroller"
	"github.com/openshift/library-go/pkg/operator/status"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openshift/cluster-kube-storage-version-migrator-operator/pkg"
	"github.com/openshift/cluster-kube-storage-version-migrator-operator/pkg/operator/assets"
	"github.com/openshift/cluster-kube-storage-version-migrator-operator/pkg/operator/deploymentcontroller"
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

	operatorClient, dynamicInformers, err := genericoperatorclient.NewClusterScopedOperatorClient(cc.KubeConfig, operatorv1.GroupVersion.WithResource("kubestorageversionmigrators"))
	if err != nil {
		return err
	}

	clusterOperator, err := configClient.ConfigV1().ClusterOperators().Get(ctx, "kube-storage-version-migrator-apiserver", metav1.GetOptions{})
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
		assets.Asset,
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

	loggingController := loglevel.NewClusterOperatorLoggingController(operatorClient, cc.EventRecorder)

	configInformers.Start(ctx.Done())
	dynamicInformers.Start(ctx.Done())
	kubeInformersForNamespaces.Start(ctx.Done())

	go statusController.Run(ctx, 1)
	go staticResourceController.Run(ctx, 1)
	go migratorDeploymentController.Run(ctx, 1)
	go loggingController.Run(ctx, 1)

	<-ctx.Done()
	return fmt.Errorf("stopped")
}
