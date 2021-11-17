package staticconditionscontroller

import (
	"context"

	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
)

type staticConditionsController struct {
	factory.Controller
}

// NewStaticConditionsController ensures the specified conditions are set as specified.
func NewStaticConditionsController(
	operatorClient v1helpers.OperatorClient, recorder events.Recorder, conditions ...operatorv1.OperatorCondition,
) *staticConditionsController {
	var updates []v1helpers.UpdateStatusFunc
	for _, condition := range conditions {
		updates = append(updates, v1helpers.UpdateConditionFn(condition))
	}
	return &staticConditionsController{
		Controller: factory.New().
			WithSync(func(ctx context.Context, controllerContext factory.SyncContext) error {
				_, _, err := v1helpers.UpdateStatus(operatorClient, updates...)
				return err
			}).
			WithInformers(operatorClient.Informer()).
			ToController("StaticConditionsController", recorder.WithComponentSuffix("static-conditions-controller")),
	}
}
