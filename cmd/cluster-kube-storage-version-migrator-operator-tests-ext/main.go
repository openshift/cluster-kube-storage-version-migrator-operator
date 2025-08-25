package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/openshift-eng/openshift-tests-extension/pkg/cmd"
	"github.com/openshift-eng/openshift-tests-extension/pkg/dbtime"
	"github.com/openshift-eng/openshift-tests-extension/pkg/extension"
	"github.com/openshift-eng/openshift-tests-extension/pkg/extension/extensiontests"

	"github.com/spf13/cobra"

	_ "github.com/openshift/cluster-kube-storage-version-migrator-operator/test/extended"
)

var (
	CommitFromGit string
	BuildDate     string
	GitTreeState  string
)

// GinkgoTestingT implements the minimal TestingT interface needed by Ginkgo
type GinkgoTestingT struct{}

func (GinkgoTestingT) Errorf(format string, args ...interface{}) {}
func (GinkgoTestingT) Fail()                                     {}
func (GinkgoTestingT) FailNow()                                  { os.Exit(1) }

// NewGinkgoTestingT creates a new testing.T compatible instance for Ginkgo
func NewGinkgoTestingT() *GinkgoTestingT {
	return &GinkgoTestingT{}
}

// escapeRegexChars escapes special regex characters in test names for Ginkgo focus
func escapeRegexChars(s string) string {
	// Only escape the problematic characters that cause regex parsing issues
	// We need to escape [ and ] which are treated as character classes
	s = strings.ReplaceAll(s, "[", "\\[")
	s = strings.ReplaceAll(s, "]", "\\]")
	return s
}

// createTestSpec creates a test spec with proper execution functions
func createTestSpec(name, source string, codeLocations []string, junitDirPtr *string) *extensiontests.ExtensionTestSpec {
	return &extensiontests.ExtensionTestSpec{
		Name:          name,
		Source:        source,
		CodeLocations: codeLocations,
		Lifecycle:     extensiontests.LifecycleBlocking,
		Resources: extensiontests.Resources{
			Isolation: extensiontests.Isolation{},
		},
		EnvironmentSelector: extensiontests.EnvironmentSelector{},
		Run: func(ctx context.Context) *extensiontests.ExtensionTestResult {
			junitDir := ""
			if junitDirPtr != nil {
				junitDir = *junitDirPtr
			}
			return runGinkgoTest(ctx, name, junitDir)
		},
		RunParallel: func(ctx context.Context) *extensiontests.ExtensionTestResult {
			junitDir := ""
			if junitDirPtr != nil {
				junitDir = *junitDirPtr
			}
			return runGinkgoTest(ctx, name, junitDir)
		},
	}
}

// runGinkgoTest runs a Ginkgo test in-process
func runGinkgoTest(ctx context.Context, testName string, junitDir string) *extensiontests.ExtensionTestResult {
	startTime := time.Now()

	// Configure Ginkgo to run specific test
	gomega.RegisterFailHandler(ginkgo.Fail)

	// Run the test suite with focus on specific test
	suiteConfig, reporterConfig := ginkgo.GinkgoConfiguration()
	suiteConfig.FocusStrings = []string{escapeRegexChars(testName)}

	// Configure JUnit reporter for CI integration only when junit-dir is provided
	if junitDir != "" {
		junitPath := filepath.Join(junitDir, "junit.xml")
		reporterConfig.JUnitReport = junitPath
		reporterConfig.JSONReport = filepath.Join(junitDir, "report.json")
	}

	passed := ginkgo.RunSpecs(NewGinkgoTestingT(), "OpenShift Kube Storage Version Migrator Operator Test Suite", suiteConfig, reporterConfig)

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	result := extensiontests.ResultPassed
	if !passed {
		result = extensiontests.ResultFailed
	}

	return &extensiontests.ExtensionTestResult{
		Name:      testName,
		Result:    result,
		StartTime: dbtime.Ptr(startTime),
		EndTime:   dbtime.Ptr(endTime),
		Duration:  int64(duration.Seconds()),
		Output:    "",
	}
}

func main() {
	// Define global flags
	var junitDir string

	// Create a new registry
	registry := extension.NewRegistry()

	// Create extension for this component
	ext := extension.NewExtension("openshift", "payload", "cluster-kube-storage-version-migrator-operator")

	// Set source information
	ext.Source = extension.Source{
		Commit:       CommitFromGit,
		BuildDate:    BuildDate,
		GitTreeState: GitTreeState,
	}

	// Add test suites
	ext.AddGlobalSuite(extension.Suite{
		Name:        "openshift/cluster-kube-storage-version-migrator-operator/conformance/parallel",
		Description: "",
		Parents:     []string{"openshift/conformance/parallel"},
		Qualifiers:  []string{"(source == \"openshift:payload:cluster-kube-storage-version-migrator-operator\") && (!(name.contains(\"[Serial]\") || name.contains(\"[Slow]\")))"},
	})

	ext.AddGlobalSuite(extension.Suite{
		Name:        "openshift/cluster-kube-storage-version-migrator-operator/conformance/serial",
		Description: "",
		Parents:     []string{"openshift/conformance/serial"},
		Qualifiers:  []string{"(source == \"openshift:payload:cluster-kube-storage-version-migrator-operator\") && (name.contains(\"[Serial]\"))"},
	})

	ext.AddGlobalSuite(extension.Suite{
		Name:        "openshift/cluster-kube-storage-version-migrator-operator/optional/slow",
		Description: "",
		Parents:     []string{"openshift/optional/slow"},
		Qualifiers:  []string{"(source == \"openshift:payload:cluster-kube-storage-version-migrator-operator\") && (name.contains(\"[Slow]\"))"},
	})

	ext.AddGlobalSuite(extension.Suite{
		Name:        "openshift/cluster-kube-storage-version-migrator-operator/all",
		Description: "",
		Qualifiers:  []string{"source == \"openshift:payload:cluster-kube-storage-version-migrator-operator\""},
	})

	// Add test specs with proper execution functions
	testSpecs := extensiontests.ExtensionTestSpecs{
		createTestSpec(
			"[Jira:storage-version-migrator][sig-api-machinery] sanity test should always pass [Suite:openshift/cluster-kube-storage-version-migrator-operator/conformance/parallel]",
			"openshift:payload:cluster-kube-storage-version-migrator-operator",
			[]string{
				"/test/extended/main.go:8",
				"/test/extended/main.go:9",
			},
			&junitDir,
		),
	}
	ext.AddSpecs(testSpecs)

	// Register the extension
	registry.Register(ext)

	// Create root command with default extension commands
	rootCmd := &cobra.Command{
		Use:   "cluster-kube-storage-version-migrator-operator-tests-ext",
		Short: "OpenShift kube-storage-version-migrator operator tests extension",
	}

	// Add global flags
	rootCmd.PersistentFlags().StringVar(&junitDir, "junit-dir", "", "Path to directory for junit.xml report")

	// Add all the default extension commands (info, list, run-test, run-suite, update)
	rootCmd.AddCommand(cmd.DefaultExtensionCommands(registry)...)

	// Execute the command
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
