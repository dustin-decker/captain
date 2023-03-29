package main

import (
	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/providers"
)

var (
	// updateCmd represents the "update" sub-command itself
	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Updates a specific resource in captain",
	}

	// updateResultsCmd is the "results" sub-command of "update".
	updateResultsCmd = &cobra.Command{
		Use:   "results [flags] --suite-id=<suite> <args>",
		Short: "Updates captain with new test-results",
		Long: "'captain update results' will parse a test-results file and updates captain's internal storage " +
			"accordingly.",
		Example: `captain update results --suite-id="JUnit" *.xml`,
		Args:    cobra.MinimumNArgs(1),
		PreRunE: initCLIService(providers.Validate),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Should also support reading from stdin
			_, err := captain.UpdateTestResults(cmd.Context(), suiteID, args)
			return errors.WithStack(err)
		},
	}
)

func init() {
	updateResultsCmd.Flags().StringVar(&githubJobName, "github-job-name", "", "the name of the current Github Job")
	if err := updateResultsCmd.Flags().MarkDeprecated("github-job-name", "the value will be ignored"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	updateResultsCmd.Flags().
		StringVar(&githubJobMatrix, "github-job-matrix", "", "the JSON encoded job-matrix from Github")
	if err := updateResultsCmd.Flags().MarkDeprecated("github-job-matrix", "the value will be ignored"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	addFrameworkFlags(updateResultsCmd)
	updateCmd.AddCommand(updateResultsCmd)
	rootCmd.AddCommand(updateCmd)
}
