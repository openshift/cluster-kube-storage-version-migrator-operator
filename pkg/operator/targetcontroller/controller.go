package targetcontroller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsv1informers "k8s.io/client-go/informers/apps/v1"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	operatorv1 "github.com/openshift/api/operator/v1"
	operatorv1client "github.com/openshift/client-go/operator/clientset/versioned/typed/operator/v1"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/status"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
)

const (
	OperatorNamespace = "openshift-kube-storage-version-migrator-operator"
	TargetNamespace   = "kube-storage-version-migrator"
	workQueueKey      = "key"
)

// TargetController syncs the resources needed to run the target operand.
type TargetController struct {
	kubeClient                  kubernetes.Interface
	genericOperatorConfigClient v1helpers.OperatorClient
	operatorConfigClient        operatorv1client.KubeStorageVersionMigratorInterface
	imagePullSpec               string
	operatorImagePullSpec       string

	eventRecorder   events.Recorder
	versionRecorder status.VersionGetter

	queue        workqueue.RateLimitingInterface
	cachesToSync []cache.InformerSynced
}

func NewTargetController(kubeClient kubernetes.Interface,
	genericOperatorConfigClient v1helpers.OperatorClient,
	operatorConfigClient operatorv1client.KubeStorageVersionMigratorInterface,
	secretInformer corev1informers.SecretInformer,
	deploymentInformer appsv1informers.DeploymentInformer,
	imagePullSpec, operatorImagePullSpec string,
	eventRecorder events.Recorder,
	versionRecorder status.VersionGetter) *TargetController {
	controller := &TargetController{
		kubeClient:                  kubeClient,
		genericOperatorConfigClient: genericOperatorConfigClient,
		operatorConfigClient:        operatorConfigClient,
		imagePullSpec:               imagePullSpec,
		operatorImagePullSpec:       operatorImagePullSpec,
		eventRecorder:               eventRecorder.WithComponentSuffix("workload-controller"),
		versionRecorder:             versionRecorder,
		queue:                       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "KubeStorageVersionMigratorOperator"),
	}

	genericOperatorConfigClient.Informer().AddEventHandler(controller.eventHandler())
	secretInformer.Informer().AddEventHandler(controller.eventHandler())
	deploymentInformer.Informer().AddEventHandler(controller.eventHandler())

	controller.cachesToSync = append(controller.cachesToSync,
		genericOperatorConfigClient.Informer().HasSynced,
		deploymentInformer.Informer().HasSynced,
		secretInformer.Informer().HasSynced,
	)

	return controller
}

func (c *TargetController) sync() error {
	operatorConfig, err := c.operatorConfigClient.Get(context.TODO(), "cluster", metav1.GetOptions{})
	if err != nil {
		return err
	}
	spec, status, objectMetaGeneration := &operatorConfig.Spec, &operatorConfig.Status, operatorConfig.ObjectMeta.Generation

	switch spec.ManagementState {
	case operatorv1.Managed:
	case operatorv1.Unmanaged:
		return nil
	case operatorv1.Removed:
		if err := c.kubeClient.CoreV1().Namespaces().Delete(context.TODO(), TargetNamespace, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			return err
		}
		return nil
	default:
	}

	forceRequeue, err := c.syncKubeStorageVersionMigrator(spec, status, objectMetaGeneration)
	if err != nil {
		return err
	}
	if forceRequeue {
		c.queue.AddRateLimited(workQueueKey)
	}

	return nil
}

func (c *TargetController) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Infof("Starting KubeStorageVersionMigratorOperator")
	defer klog.Infof("Shutting down KubeStorageVersionMigratorOperator")
	if !cache.WaitForCacheSync(stopCh, c.cachesToSync...) {
		return
	}

	// doesn't matter what workers say, only start one.
	go wait.Until(c.runWorker, time.Second, stopCh)

	<-stopCh
}

func (c *TargetController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *TargetController) processNextWorkItem() bool {
	// get next key in the queue
	key, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	// unblock key when done
	defer c.queue.Done(key)
	// controller sync
	if err := c.sync(); err != nil {
		utilruntime.HandleError(fmt.Errorf("%v failed with : %v", key, err))
		c.queue.AddRateLimited(key)
		return true
	}
	// no need to retry this key
	c.queue.Forget(key)
	return true
}

func (c *TargetController) eventHandler() cache.ResourceEventHandler {
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { c.queue.Add(workQueueKey) },
		UpdateFunc: func(old, new interface{}) { c.queue.Add(workQueueKey) },
		DeleteFunc: func(obj interface{}) { c.queue.Add(workQueueKey) },
	}
}
