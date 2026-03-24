package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/component-base/cli"

	otecmd "github.com/openshift-eng/openshift-tests-extension/pkg/cmd"
	oteextension "github.com/openshift-eng/openshift-tests-extension/pkg/extension"
	oteginkgo "github.com/openshift-eng/openshift-tests-extension/pkg/ginkgo"
	"github.com/openshift/cluster-kube-storage-version-migrator-operator/pkg/version"

	"k8s.io/klog/v2"
)

func main() {
	cmd, err := newOperatorTestCommand()
	if err != nil {
		klog.Fatal(err)
	}
	code := cli.Run(cmd)
	os.Exit(code)
}

func newOperatorTestCommand() (*cobra.Command, error) {
	registry, err := prepareOperatorTestsRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to prepare test registry: %w", err)
	}

	cmd := &cobra.Command{
		Use:   "cluster-kube-storage-version-migrator-operator-tests-ext",
		Short: "A binary used to run cluster-kube-storage-version-migrator-operator tests as part of OTE.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				klog.Fatal(err)
			}
		},
	}

	if v := version.Get().String(); len(v) == 0 {
		cmd.Version = "<unknown>"
	} else {
		cmd.Version = v
	}

	cmd.AddCommand(otecmd.DefaultExtensionCommands(registry)...)

	return cmd, nil
}

func prepareOperatorTestsRegistry() (*oteextension.Registry, error) {
	registry := oteextension.NewRegistry()
	extension := oteextension.NewExtension("openshift", "payload", "cluster-kube-storage-version-migrator-operator")

	extension.AddSuite(oteextension.Suite{
		Name:        "openshift/cluster-kube-storage-version-migrator-operator/operator/disruptive",
		Parallelism: 1,
		Qualifiers: []string{
			`name.contains("[Disruptive]") && name.contains("[Serial]")`,
		},
		ClusterStability: oteextension.ClusterStabilityDisruptive,
	})

	extension.AddSuite(oteextension.Suite{
		Name: "openshift/cluster-kube-storage-version-migrator-operator/operator/parallel",
		Qualifiers: []string{
			`!name.contains("[Serial]") && !name.contains("[Disruptive]")`,
		},
	})

	specs, err := oteginkgo.BuildExtensionTestSpecsFromOpenShiftGinkgoSuite()
	if err != nil {
		return nil, fmt.Errorf("couldn't build extension test specs from ginkgo: %w", err)
	}

	extension.AddSpecs(specs)
	registry.Register(extension)
	return registry, nil
}
