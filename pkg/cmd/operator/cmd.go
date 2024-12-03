package operator

import (
	"context"

	"github.com/spf13/cobra"

	"k8s.io/utils/clock"

	"github.com/openshift/cluster-kube-storage-version-migrator-operator/pkg"
	"github.com/openshift/cluster-kube-storage-version-migrator-operator/pkg/operator"
	"github.com/openshift/cluster-kube-storage-version-migrator-operator/pkg/version"
	"github.com/openshift/library-go/pkg/controller/controllercmd"
)

func NewOperator() *cobra.Command {
	cmd := controllercmd.NewControllerCommandConfig(
		pkg.OperatorNamespace,
		version.Get(),
		operator.RunOperator,
		clock.RealClock{},
	).NewCommandWithContext(context.TODO())
	cmd.Use = "start"
	cmd.Short = "Start the Cluster Storage Version Migrator Operator"
	return cmd
}
