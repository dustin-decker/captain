//go:build integration

package integration_test

import (
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cloud Mode Integration Tests", func() {
	BeforeEach(func() {
		Expect(os.Getenv("RWX_ACCESS_TOKEN")).ToNot(BeEmpty(), "These integration tests require a valid RWX_ACCESS_TOKEN")
	})

	withAndWithoutInheritedEnv(func(getEnv envGenerator, prefix string) {
		getEnvWithAccessToken := func() map[string]string {
			env := getEnv()
			env["RWX_ACCESS_TOKEN"] = os.Getenv("RWX_ACCESS_TOKEN")
			return env
		}
		Describe("captain run", func() {
			Context("quarantining", func() {
				It("succeeds when all failures quarantined", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-quarantine-test",
							"--test-results", "fixtures/integration-tests/rspec-quarantine.json",
							"--fail-on-upload-error",
							"-c", "bash -c 'exit 2'",
						},
						env: getEnvWithAccessToken(),
					})

					Expect(result.stderr).To(BeEmpty())
					Expect(result.exitCode).To(Equal(0))
				})

				It("fails & passes through exit code when not all failures quarantined", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-quarantine-test",
							"--test-results", "fixtures/integration-tests/rspec-quarantined-with-other-errors.json",
							"--fail-on-upload-error",
							"-c", "bash -c 'exit 123'",
						},
						env: getEnvWithAccessToken(),
					})

					Expect(result.stderr).To(Equal("Error: test suite exited with non-zero exit code"))
					Expect(result.exitCode).To(Equal(123))
				})
			})

			Context("retries", func() {
				var _symlinkDestPath string
				var _symlinkSrcPath string

				// retry tests delete test results between retries.
				// this function ensures a symlink exists to the test results file
				// that can be freely removed
				// the symlink will be resuscitated after the test in the AfterEach
				symlinkToNewPath := func(srcPath string, prefix string) string {
					var err error
					_symlinkDestPath = fmt.Sprintf("fixtures/integration-tests/retries/%s-%s", prefix, filepath.Base(srcPath))
					_symlinkSrcPath = fmt.Sprintf("../%s", filepath.Base(srcPath))
					Expect(err).ToNot(HaveOccurred())

					os.Symlink(_symlinkSrcPath, _symlinkDestPath)
					return _symlinkDestPath
				}

				AfterEach(func() {
					os.Symlink(_symlinkSrcPath, _symlinkDestPath)
				})

				It("succeeds when all failures quarantined", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-quarantine-test",
							"--test-results", symlinkToNewPath("fixtures/integration-tests/rspec-quarantine.json", prefix),
							"--fail-on-upload-error",
							"--retries", "1",
							"--retry-command", `echo "{{ tests }}"`,
							"-c", "bash -c 'exit 123'",
						},
						env: getEnvWithAccessToken(),
					})

					Expect(result.stderr).To(BeEmpty())
					Expect(result.stdout).To(ContainSubstring("'./x.rb[1:1]'"))
					Expect(result.exitCode).To(Equal(0))
				})

				It("fails & passes through exit code on failure", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-functional-tests",
							"--test-results", symlinkToNewPath("fixtures/integration-tests/rspec-failed-not-quarantined.json", prefix),
							"--fail-on-upload-error",
							"--retries", "1",
							"--retry-command", `echo "{{ tests }}"`,
							"-c", "bash -c 'exit 123'",
						},
						env: getEnvWithAccessToken(),
					})

					Expect(result.stderr).To(Equal("Error: test suite exited with non-zero exit code"))
					Expect(result.stdout).To(ContainSubstring("'./x.rb[1:1]'"))
					Expect(result.exitCode).To(Equal(123))
				})
			})

			Context("with abq", func() {
				It("runs with ABQ_SET_EXIT_CODE=false when ABQ_SET_EXIT_CODE is unset", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-abq-test",
							"--test-results", "fixtures/integration-tests/rspec-quarantine.json",
							"--fail-on-upload-error",
							"-c", "bash -c 'echo exit_code=$ABQ_SET_EXIT_CODE'",
						},
						env: getEnvWithAccessToken(),
					})

					Expect(result.stderr).To(BeEmpty())
					Expect(result.stdout).To(HavePrefix("exit_code=false"))
					Expect(result.exitCode).To(Equal(0))
				})

				It("runs with ABQ_SET_EXIT_CODE=false when ABQ_SET_EXIT_CODE is already set", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-abq-test",
							"--test-results", "fixtures/integration-tests/rspec-quarantine.json",
							"--fail-on-upload-error",
							"-c", "bash -c 'echo exit_code=$ABQ_SET_EXIT_CODE'",
						},
						env: mergeMaps(getEnv(), map[string]string{"ABQ_SET_EXIT_CODE": "1234"}),
					})

					Expect(result.stdout).To(HavePrefix("exit_code=false"))
					Expect(result.exitCode).To(Equal(0))
				})

				It("runs with new ABQ_STATE_FILE path when ABQ_STATE_FILE is unset", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-abq-test",
							"--test-results", "fixtures/integration-tests/rspec-quarantine.json",
							"--fail-on-upload-error",
							"-c", "bash -c 'echo state_file=$ABQ_STATE_FILE'",
						},
						env: getEnvWithAccessToken(),
					})

					Expect(result.stderr).To(BeEmpty())
					Expect(result.stdout).To(HavePrefix("state_file=/tmp/captain-abq-"))
					Expect(result.exitCode).To(Equal(0))
				})

				It("runs with previously set ABQ_STATE_FILE path when ABQ_STATE_FILE is set", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-abq-test",
							"--test-results", "fixtures/integration-tests/rspec-quarantine.json",
							"--fail-on-upload-error",
							"-c", "bash -c 'echo state_file=$ABQ_STATE_FILE'",
						},
						env: mergeMaps(getEnv(), map[string]string{"ABQ_STATE_FILE": "/tmp/functional-abq-1234.json"}),
					})

					Expect(result.stdout).To(HavePrefix("state_file=/tmp/functional-abq-1234.json"))
					Expect(result.exitCode).To(Equal(0))
				})
			})
		})

		Describe("captain quarantine", func() {
			It("succeeds when all failures quarantined", func() {
				result := runCaptain(captainArgs{
					args: []string{
						"quarantine",
						"--suite-id", "captain-cli-quarantine-test",
						"--test-results", "fixtures/integration-tests/rspec-quarantine.json",
						"-c", "bash -c 'exit 2'",
					},
					env: getEnvWithAccessToken(),
				})

				Expect(result.stderr).To(BeEmpty())
				Expect(result.exitCode).To(Equal(0))
			})

			It("fails & passes through exit code when not all failures quarantined", func() {
				result := runCaptain(captainArgs{
					args: []string{
						"quarantine",
						"--suite-id", "captain-cli-quarantine-test",
						"--test-results", "fixtures/integration-tests/rspec-quarantined-with-other-errors.json",
						"-c", "bash -c 'exit 123'",
					},
					env: getEnvWithAccessToken(),
				})

				Expect(result.stderr).To(Equal("Error: test suite exited with non-zero exit code"))
				Expect(result.exitCode).To(Equal(123))
			})
		})

		Describe("captain partition", func() {
			Context("without timings", func() {
				It("sets partition 1 correctly", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"partition",
							"fixtures/integration-tests/partition/x.rb",
							"fixtures/integration-tests/partition/y.rb",
							"fixtures/integration-tests/partition/z.rb",
							"--suite-id", "captain-cli-functional-tests",
							"--index", "0",
							"--total", "2",
						},
						env: getEnvWithAccessToken(),
					})

					Expect(result.stderr).To(ContainSubstring("No test file timings were matched."))
					Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/x.rb fixtures/integration-tests/partition/z.rb"))
					Expect(result.exitCode).To(Equal(0))
				})

				It("sets partition 2 correctly", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"partition",
							"fixtures/integration-tests/partition/x.rb",
							"fixtures/integration-tests/partition/y.rb",
							"fixtures/integration-tests/partition/z.rb",
							"--suite-id", "captain-cli-functional-tests",
							"--index", "1",
							"--total", "2",
						},
						env: getEnvWithAccessToken(),
					})

					Expect(result.stderr).To(ContainSubstring("No test file timings were matched."))
					Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/y.rb"))
					Expect(result.exitCode).To(Equal(0))
				})
			})

			Context("with timings", func() {
				// to regenerate timings, edit rspec-partition.json and then run
				// 1. captain upload results test/fixtures/integration-tests/partition/rspec-partition.json --suite-id captain-cli-functional-tests
				// 2. change the CAPTAIN_SHA parameter to pull in the new timings

				It("sets partition 1 correctly", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"partition",
							"fixtures/integration-tests/partition/*_spec.rb",
							"--suite-id", "captain-cli-functional-tests",
							"--index", "0",
							"--total", "2",
						},
						env: getEnvWithAccessToken(),
					})

					Expect(result.stderr).To(BeEmpty())
					Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/a_spec.rb fixtures/integration-tests/partition/d_spec.rb"))
					Expect(result.exitCode).To(Equal(0))
				})

				It("sets partition 2 correctly", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"partition",
							"fixtures/integration-tests/partition/*_spec.rb",
							"--suite-id", "captain-cli-functional-tests",
							"--index", "1",
							"--total", "2",
						},
						env: getEnvWithAccessToken(),
					})

					Expect(result.stderr).To(BeEmpty())
					Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/b_spec.rb fixtures/integration-tests/partition/c_spec.rb"))
					Expect(result.exitCode).To(Equal(0))
				})
			})
		})

		Describe("captain upload", func() {
			It("short circuits when there's nothing to upload", func() {
				result := runCaptain(captainArgs{
					args: []string{
						"upload", "results",
						"nonexistingfile.json",
						"--suite-id", "captain-cli-functional-tests",
					},
					env: getEnvWithAccessToken(),
				})

				Expect(result.stderr).To(BeEmpty())
				Expect(result.stdout).To(BeEmpty())
				Expect(result.exitCode).To(Equal(0))
			})
		})
	})
})