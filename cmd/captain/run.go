package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/providers"
	"github.com/rwx-research/captain-cli/internal/reporting"
	"github.com/rwx-research/captain-cli/internal/targetedretries"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type CliArgs struct {
	testResults               string
	failOnUploadError         bool
	failRetriesFast           bool
	flakyRetries              int
	intermediateArtifactsPath string
	maxTestsToRetry           string
	postRetryCommands         []string
	preRetryCommands          []string
	printSummary              bool
	quiet                     bool
	reporters                 []string
	retries                   int
	retryCommandTemplate      string
	updateStoredResults       bool
	GenericProvider           providers.GenericEnv
}

var (
	cliArgs                  CliArgs
	substitutionsByFramework = map[v1.Framework]targetedretries.Substitution{
		v1.DotNetxUnitFramework:          new(targetedretries.DotNetxUnitSubstitution),
		v1.ElixirExUnitFramework:         new(targetedretries.ElixirExUnitSubstitution),
		v1.GoGinkgoFramework:             new(targetedretries.GoGinkgoSubstitution),
		v1.GoTestFramework:               new(targetedretries.GoTestSubstitution),
		v1.JavaScriptCypressFramework:    new(targetedretries.JavaScriptCypressSubstitution),
		v1.JavaScriptJestFramework:       new(targetedretries.JavaScriptJestSubstitution),
		v1.JavaScriptMochaFramework:      new(targetedretries.JavaScriptMochaSubstitution),
		v1.JavaScriptPlaywrightFramework: new(targetedretries.JavaScriptPlaywrightSubstitution),
		v1.PHPUnitFramework:              new(targetedretries.PHPUnitSubstitution),
		v1.PythonPytestFramework:         new(targetedretries.PythonPytestSubstitution),
		v1.PythonUnitTestFramework:       new(targetedretries.PythonUnitTestSubstitution),
		v1.RubyCucumberFramework:         new(targetedretries.RubyCucumberSubstitution),
		v1.RubyMinitestFramework:         new(targetedretries.RubyMinitestSubstitution),
		v1.RubyRSpecFramework:            new(targetedretries.RubyRSpecSubstitution),
	}

	runCmd = &cobra.Command{
		Use:   "run [flags] --suite-id=<suite> <args>",
		Short: "Execute a build- or test-suite",
		Long:  "'captain run' can be used to execute a build- or test-suite and optionally upload the resulting artifacts.",
		Example: `  captain run --suite-id="your-project-rake" -- bundle exec rake` + "\n" +
			`  captain run --suite-id="your-project-jest" --test-results "jest-result.json" -- jest`,
		Args:    cobra.MinimumNArgs(1),
		PreRunE: initCLIService(providers.Validate),
		RunE: func(cmd *cobra.Command, args []string) error {
			var postRetryCommands, preRetryCommands []string
			var failOnUploadError, failFast, printSummary, quiet bool
			var flakyRetries, retries int
			var intermediateArtifactsPath, retryCommand, testResultsPath, maxTests string

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

				failOnUploadError = suiteConfig.FailOnUploadError
				failFast = suiteConfig.Retries.FailFast
				flakyRetries = suiteConfig.Retries.FlakyAttempts
				maxTests = suiteConfig.Retries.MaxTests
				postRetryCommands = suiteConfig.Retries.PostRetryCommands
				preRetryCommands = suiteConfig.Retries.PreRetryCommands
				printSummary = suiteConfig.Output.PrintSummary
				quiet = suiteConfig.Output.Quiet
				retries = suiteConfig.Retries.Attempts
				retryCommand = suiteConfig.Retries.Command
				intermediateArtifactsPath = suiteConfig.Retries.IntermediateArtifactsPath
				testResultsPath = os.ExpandEnv(suiteConfig.Results.Path)
			}

			runConfig := cli.RunConfig{
				Args:                      args,
				FailOnUploadError:         failOnUploadError,
				FailRetriesFast:           failFast,
				FlakyRetries:              flakyRetries,
				IntermediateArtifactsPath: intermediateArtifactsPath,
				MaxTestsToRetry:           maxTests,
				PostRetryCommands:         postRetryCommands,
				PreRetryCommands:          preRetryCommands,
				PrintSummary:              printSummary,
				Quiet:                     quiet,
				Reporters:                 reporterFuncs,
				Retries:                   retries,
				RetryCommandTemplate:      retryCommand,
				SubstitutionsByFramework:  substitutionsByFramework,
				SuiteID:                   suiteID,
				TestResultsFileGlob:       testResultsPath,
				UpdateStoredResults:       cliArgs.updateStoredResults,
				UploadResults:             true,
			}

			err := captain.RunSuite(cmd.Context(), runConfig)
			if _, ok := errors.AsConfigurationError(err); !ok {
				cmd.SilenceUsage = true
			}

			return errors.WithDecoration(err)
		},
	}
)

func AddFlags(runCmd *cobra.Command, cliArgs *CliArgs) {
	runCmd.Flags().StringVar(
		&cliArgs.testResults,
		"test-results",
		"",
		"a filepath to a test result - supports globs for multiple result files",
	)

	runCmd.Flags().BoolVar(
		&cliArgs.failOnUploadError,
		"fail-on-upload-error",
		false,
		"return a non-zero exit code in case the test results upload fails",
	)

	runCmd.Flags().StringVar(
		&cliArgs.intermediateArtifactsPath,
		"intermediate-artifacts-path",
		"",
		"the path to store intermediate artifacts under. Intermediate artifacts will be removed if not set.",
	)

	runCmd.Flags().StringArrayVar(
		&cliArgs.postRetryCommands,
		"post-retry",
		[]string{},
		"commands to run immediately after captain retries a test",
	)

	runCmd.Flags().StringArrayVar(
		&cliArgs.preRetryCommands,
		"pre-retry",
		[]string{},
		"commands to run immediately before captain retries a test",
	)

	runCmd.Flags().BoolVar(
		&cliArgs.printSummary,
		"print-summary",
		false,
		"prints a summary of all tests to the console",
	)

	runCmd.Flags().StringArrayVar(
		&cliArgs.reporters,
		"reporter",
		[]string{},
		"one or more `type=output_path` pairs to enable different reporting options. "+
			"Available reporter types are `rwx-v1-json` and `junit-xml ",
	)

	runCmd.Flags().IntVar(
		&cliArgs.retries,
		"retries",
		-1,
		"the number of times failed tests should be retried "+
			"(e.g. --retries 2 would mean a maximum of 3 attempts of any given test)",
	)

	runCmd.Flags().IntVar(
		&cliArgs.flakyRetries,
		"flaky-retries",
		-1,
		"the number of times failing flaky tests should be retried (takes precedence over --retries if the test is known "+
			"to be flaky) (e.g. --flaky-retries 2 would mean a maximum of 3 attempts of any flaky test)",
	)

	runCmd.Flags().StringVar(
		&cliArgs.maxTestsToRetry,
		"max-tests-to-retry",
		"",
		"if set, retries will not be run when there are more than N tests to retry or if more than N%% of all tests "+
			"need retried (e.g. --max-tests-to-retry 15 or --max-tests-to-retry 1.5%)",
	)

	runCmd.Flags().BoolVar(
		&cliArgs.failRetriesFast,
		"fail-retries-fast",
		false,
		"if set, your test suite will fail as quickly as we know it will fail (e.g. with --retries 1 and "+
			"--flaky-retries 5, you might have a non-flaky test that we stop retrying after 1 additional attempt. "+
			"in this situation, we know the tests overall will fail so we can stop retrying to save compute. similarly "+
			"if you only set --flaky-retries 1, we can stop retrying if any non-flaky tests fail because we won't retry "+
			"them)",
	)

	runCmd.Flags().StringVar(&githubJobName, "github-job-name", "", "the name of the current Github Job")
	if err := runCmd.Flags().MarkDeprecated("github-job-name", "the value will be ignored"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}
	runCmd.Flags().StringVar(&githubJobMatrix, "github-job-matrix", "", "the JSON encoded job-matrix from Github")
	if err := runCmd.Flags().MarkDeprecated("github-job-matrix", "the value will be ignored"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	formattedSubstitutionExamples := make([]string, len(substitutionsByFramework))
	i := 0
	for framework, substitution := range substitutionsByFramework {
		formattedSubstitutionExamples[i] = fmt.Sprintf("  %v: --retry-command \"%v\"", framework, substitution.Example())
		i++
	}
	sort.SliceStable(formattedSubstitutionExamples, func(i, j int) bool {
		return strings.ToLower(formattedSubstitutionExamples[i]) < strings.ToLower(formattedSubstitutionExamples[j])
	})

	runCmd.Flags().StringVar(
		&cliArgs.retryCommandTemplate,
		"retry-command",
		"",
		fmt.Sprintf(
			"the command that will be run to execute a subset of your tests while retrying "+
				"(required if --retries or --flaky-retries is passed)\n"+
				"Examples:\n  Custom: --retry-command \"%v\"\n%v",
			targetedretries.JSONSubstitution{}.Example(),
			strings.Join(formattedSubstitutionExamples, "\n"),
		),
	)

	runCmd.Flags().BoolVar(
		&cliArgs.updateStoredResults,
		"update-stored-results",
		false,
		"if set, captain will update its internal storage files under '.captain' with the latest test results, "+
			"such as flaky tests and test timings.",
	)

	addGenericProviderFlags(runCmd, &cliArgs.GenericProvider)
	addFrameworkFlags(runCmd)
}

func init() {
	AddFlags(runCmd, &cliArgs)
	runCmd.SetHelpTemplate(helpTemplate)
	runCmd.SetUsageTemplate(shortUsageTemplate)
	rootCmd.AddCommand(runCmd)
}

// this should be run _last_ as it has the highest precedence, and the assignments we make here overwrite settings
// from other parts of the app (e.g. config files, env vars)
func bindRunCmdFlags(cfg Config) Config {
	if suiteConfig, ok := cfg.TestSuites[suiteID]; ok {
		if cliArgs.failOnUploadError {
			suiteConfig.FailOnUploadError = true
		}

		if cliArgs.testResults != "" {
			suiteConfig.Results.Path = cliArgs.testResults
		}

		if len(cliArgs.postRetryCommands) != 0 {
			suiteConfig.Retries.PostRetryCommands = cliArgs.postRetryCommands
		}

		if len(cliArgs.preRetryCommands) != 0 {
			suiteConfig.Retries.PreRetryCommands = cliArgs.preRetryCommands
		}

		if cliArgs.failRetriesFast {
			suiteConfig.Retries.FailFast = true
		}

		// We want to use the default as set by `cobra`
		if suiteConfig.Retries.FlakyAttempts == 0 || cliArgs.flakyRetries != -1 {
			suiteConfig.Retries.FlakyAttempts = cliArgs.flakyRetries
		}

		if cliArgs.maxTestsToRetry != "" {
			suiteConfig.Retries.MaxTests = cliArgs.maxTestsToRetry
		}

		if cliArgs.printSummary {
			suiteConfig.Output.PrintSummary = true
		}

		if cliArgs.quiet {
			suiteConfig.Output.Quiet = true
		}

		if cliArgs.reporters != nil {
			reporterConfig := make(map[string]string)

			for _, r := range cliArgs.reporters {
				name, path, _ := strings.Cut(r, "=")
				reporterConfig[name] = path
			}

			suiteConfig.Output.Reporters = reporterConfig
		}

		if suiteConfig.Retries.Attempts == 0 || cliArgs.retries != -1 {
			suiteConfig.Retries.Attempts = cliArgs.retries
		}

		if cliArgs.retryCommandTemplate != "" {
			suiteConfig.Retries.Command = cliArgs.retryCommandTemplate
		}

		if cliArgs.intermediateArtifactsPath != "" {
			suiteConfig.Retries.IntermediateArtifactsPath = cliArgs.intermediateArtifactsPath
		}

		cfg.TestSuites[suiteID] = suiteConfig

		cfg.ProvidersEnv.Generic = providers.MergeGeneric(cfg.ProvidersEnv.Generic, cliArgs.GenericProvider)
	}

	return cfg
}

func addGenericProviderFlags(cmd *cobra.Command, destination *providers.GenericEnv) {
	cmd.Flags().StringVar(
		&destination.Branch,
		"branch",
		os.Getenv("CAPTAIN_BRANCH"),
		"the branch name of the commit being built\n"+
			"if using a supported CI provider, this will be automatically set\n"+
			"otherwise use this flag or set the environment variable CAPTAIN_BRANCH\n",
	)

	cmd.Flags().StringVar(
		&destination.Who,
		"who",
		os.Getenv("CAPTAIN_WHO"),
		"the person who triggered the build\n"+
			"if using a supported CI provider, this will be automatically set\n"+
			"otherwise use this flag or set the environment variable CAPTAIN_WHO\n",
	)

	addShaFlag(cmd, &destination.Sha)

	cmd.Flags().StringVar(
		&destination.CommitMessage,
		"commit-message",
		os.Getenv("CAPTAIN_COMMIT_MESSAGE"),
		"the git commit message of the commit being built\n"+
			"if using a supported CI provider, this will be automatically set\n"+
			"otherwise use this flag or set the environment variable CAPTAIN_COMMIT_MESSAGE\n",
	)

	cmd.Flags().StringVar(
		&destination.BuildURL,
		"build-url",
		os.Getenv("CAPTAIN_BUILD_URL"),
		"the URL of the build results\n"+
			"if using a supported CI provider, this will be automatically set\n"+
			"otherwise use this flag or set the environment variable CAPTAIN_BUILD_URL\n",
	)
}

func addShaFlag(cmd *cobra.Command, destination *string) {
	cmd.Flags().StringVar(
		destination,
		"sha",
		os.Getenv("CAPTAIN_SHA"),
		"the git commit sha hash of the commit being built",
	)
}
