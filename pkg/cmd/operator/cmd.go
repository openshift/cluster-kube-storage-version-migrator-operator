package operator

import (
	"context"

	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/spf13/cobra"

	"github.com/openshift/cluster-kube-storage-version-migrator-operator/pkg"
	"github.com/openshift/cluster-kube-storage-version-migrator-operator/pkg/operator"
	"github.com/openshift/cluster-kube-storage-version-migrator-operator/pkg/version"
)

func NewOperator() *cobra.Command {
	cmd := controllercmd.NewControllerCommandConfig(
		pkg.OperatorNamespace,
		version.Get(),
		operator.RunOperator,
	).NewCommandWithContext(context.TODO())
	cmd.Use = "start"
	cmd.Short = "Start the Cluster Storage Version Migrator Operator"
	return cmd
}
