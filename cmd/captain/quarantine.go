package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/providers"
	"github.com/rwx-research/captain-cli/internal/reporting"
)

var quarantineCmd = &cobra.Command{
	Use:   "quarantine [flags] --suite-id=<suite> <args>",
	Short: "Execute a test-suite and modify its exit code based on quarantined tests",
	Long: "'captain quarantine' executes a test-suite and modifies its exit code based on quarantined tests." +
		"Unlike run, it does not attempt retries or update test results.",
	Example: `  captain quarantine --suite-id "example" --test-results "./tmp/rspec.json" -- bundle exec rake`,
	Args:    cobra.MinimumNArgs(1),
	PreRunE: initCLIService(providers.Validate),
	RunE: func(cmd *cobra.Command, args []string) error {
		var printSummary, quiet bool
		var testResultsPath string

		reporterFuncs := make(map[string]cli.Reporter)

		if suiteConfig, ok := cfg.TestSuites[suiteID]; ok {
			for name, path := range suiteConfig.Output.Reporters {
				switch name {
				case "rwx-v1-json":
					reporterFuncs[path] = reporting.WriteJSONSummary
				case "junit-xml":
					reporterFuncs[path] = reporting.WriteJUnitSummary
				default:
					return errors.WithDecoration(errors.NewConfigurationError(
						fmt.Sprintf("Unknown reporter %q", name),
						"Available reporters are 'rwx-v1-json' and 'junit-xml'.",
						"",
					))
				}
			}

			printSummary = suiteConfig.Output.PrintSummary
			testResultsPath = os.ExpandEnv(suiteConfig.Results.Path)
			quiet = suiteConfig.Output.Quiet
		}

		runConfig := cli.RunConfig{
			Args:                args,
			PrintSummary:        printSummary,
			Quiet:               quiet,
			Reporters:           reporterFuncs,
			SuiteID:             suiteID,
			TestResultsFileGlob: testResultsPath,
			UpdateStoredResults: cliArgs.updateStoredResults,

			FailOnUploadError: false,
			FailRetriesFast:   false,
			FlakyRetries:      0,
			Retries:           0,
			UploadResults:     false,
		}

		err := captain.RunSuite(cmd.Context(), runConfig)
		if _, ok := errors.AsConfigurationError(err); !ok {
			cmd.SilenceUsage = true
		}

		return errors.WithDecoration(err)
	},
}

func AddQuarantineFlags(quarantineCmd *cobra.Command, cliArgs *CliArgs) {
	quarantineCmd.Flags().StringVar(
		&cliArgs.testResults,
		"test-results",
		"",
		"a filepath to a test result - supports globs for multiple result files",
	)

	quarantineCmd.Flags().BoolVarP(
		&cliArgs.quiet,
		"quiet",
		"q",
		false,
		"disables most default output",
	)

	quarantineCmd.Flags().BoolVar(
		&cliArgs.printSummary,
		"print-summary",
		false,
		"prints a summary of all tests to the console",
	)

	quarantineCmd.Flags().StringArrayVar(
		&cliArgs.reporters,
		"reporter",
		[]string{},
		"one or more `type=output_path` pairs to enable different reporting options. "+
			"Available reporter types are `rwx-v1-json` and `junit-xml ",
	)

	quarantineCmd.Flags().BoolVar(
		&cliArgs.updateStoredResults,
		"update-stored-results",
		false,
		"if set, captain will update its internal storage files under '.captain' with the latest test results, "+
			"such as flaky tests and test timings.",
	)

	addFrameworkFlags(quarantineCmd)
}

func init() {
	AddQuarantineFlags(quarantineCmd, &cliArgs)
	rootCmd.AddCommand(quarantineCmd)
}