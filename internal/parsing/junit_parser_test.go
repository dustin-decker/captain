package parsing_test

import (
	"os"
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JUnitParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/junit.xml")
			Expect(err).ToNot(HaveOccurred())

			parseResult, err := parsing.JUnitParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())

			Expect(parseResult.Parser).To(Equal(parsing.JUnitParser{}))
			Expect(parseResult.Sentiment).To(Equal(parsing.NeutralParseResultSentiment))
			Expect(parseResult.TestResults.Framework.Language).To(Equal(v1.FrameworkLanguageOther))
			Expect(parseResult.TestResults.Framework.Kind).To(Equal(v1.FrameworkKindOther))
			Expect(parseResult.TestResults.Summary.Tests).To(Equal(72))
			Expect(parseResult.TestResults.Summary.Successful).To(Equal(67))
			Expect(parseResult.TestResults.Summary.Failed).To(Equal(3))
			Expect(parseResult.TestResults.Summary.Skipped).To(Equal(2))
			Expect(parseResult.TestResults.Summary.OtherErrors).To(Equal(0))
		})

		It("errors on malformed XML", func() {
			parseResult, err := parsing.JUnitParser{}.Parse(strings.NewReader(`<abc`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(parseResult).To(BeNil())
		})

		It("errors on XML that doesn't look like JUnit", func() {
			var parseResult *parsing.ParseResult
			var err error

			parseResult, err = parsing.JUnitParser{}.Parse(strings.NewReader(`<foo></foo>`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(parseResult).To(BeNil())

			parseResult, err = parsing.JUnitParser{}.Parse(strings.NewReader(`<testsuites></testsuites>`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The test suites in the XML do not appear to match JUnit XML"))
			Expect(parseResult).To(BeNil())

			parseResult, err = parsing.JUnitParser{}.Parse(
				strings.NewReader(`<testsuites><testsuite></testsuite></testsuites>`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("The test suites in the XML do not appear to match JUnit XML"),
			)
			Expect(parseResult).To(BeNil())
		})

		It("extracts the file from a test case", func() {
			parseResult, err := parsing.JUnitParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite tests="1">
							<testcase
								name="some test name"
								classname="prefix some test name"
								file="some/path/to/file.js"
								line="12"
							>
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())
			line := 12
			Expect(*parseResult.TestResults.Tests[0].Location).To(Equal(
				v1.Location{File: "some/path/to/file.js", Line: &line},
			))
		})

		It("parses the duration as seconds", func() {
			parseResult, err := parsing.JUnitParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite tests="1">
							<testcase
								name="some test name"
								classname="prefix some test name"
								time="1.5249"
							>
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())
			Expect(*parseResult.TestResults.Tests[0].Attempt.Duration).To(Equal(
				time.Duration(1524900000),
			))
		})

		It("parses failures with inner CDATA", func() {
			parseResult, err := parsing.JUnitParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite tests="1">
							<testcase
								name="some test name"
								classname="prefix some test name"
								time="1.5249"
							>
								<failure type="someclass" message="some message"><![CDATA[line 1
									line 2

									line 3]]></failure>
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())
			message := "some message"
			exception := "someclass"
			backtrace := []string{
				"line 1",
				"line 2",
				"",
				"line 3",
			}
			Expect(parseResult.TestResults.Tests[0].Attempt.Status).To(Equal(
				v1.NewFailedTestStatus(&message, &exception, backtrace),
			))
		})

		It("parses failures with inner chardata", func() {
			parseResult, err := parsing.JUnitParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite tests="1">
							<testcase
								name="some test name"
								classname="prefix some test name"
								time="1.5249"
							>
								<failure type="someclass" message="some message">line 1
									line 2

									line 3</failure>
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())
			message := "some message"
			exception := "someclass"
			backtrace := []string{
				"line 1",
				"line 2",
				"",
				"line 3",
			}
			Expect(parseResult.TestResults.Tests[0].Attempt.Status).To(Equal(
				v1.NewFailedTestStatus(&message, &exception, backtrace),
			))
		})

		It("parses errors with inner CDATA", func() {
			parseResult, err := parsing.JUnitParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite tests="1">
							<testcase
								name="some test name"
								classname="prefix some test name"
								time="1.5249"
							>
								<error type="someclass" message="some message"><![CDATA[line 1
									line 2

									line 3]]></error>
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())
			message := "some message"
			exception := "someclass"
			backtrace := []string{
				"line 1",
				"line 2",
				"",
				"line 3",
			}
			Expect(parseResult.TestResults.Tests[0].Attempt.Status).To(Equal(
				v1.NewFailedTestStatus(&message, &exception, backtrace),
			))
		})

		It("parses errors with inner chardata", func() {
			parseResult, err := parsing.JUnitParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite tests="1">
							<testcase
								name="some test name"
								classname="prefix some test name"
								time="1.5249"
							>
								<error type="someclass" message="some message">line 1
									line 2

									line 3</error>
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())
			message := "some message"
			exception := "someclass"
			backtrace := []string{
				"line 1",
				"line 2",
				"",
				"line 3",
			}
			Expect(parseResult.TestResults.Tests[0].Attempt.Status).To(Equal(
				v1.NewFailedTestStatus(&message, &exception, backtrace),
			))
		})

		It("parses skipped tests with messages", func() {
			parseResult, err := parsing.JUnitParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite tests="1">
							<testcase
								name="some test name"
								classname="prefix some test name"
								time="1.5249"
							>
								<skipped message="some reason" />
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())
			message := "some reason"
			Expect(parseResult.TestResults.Tests[0].Attempt.Status).To(Equal(
				v1.NewSkippedTestStatus(&message),
			))
		})

		It("calculates the correct name when the classname contains the name", func() {
			parseResult, err := parsing.JUnitParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite tests="1">
							<testcase name="some test name" classname="prefix some test name">
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())
			Expect(parseResult.TestResults.Tests[0].Name).To(Equal("prefix some test name"))
		})

		It("calculates the correct name when the name contains the classname", func() {
			parseResult, err := parsing.JUnitParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite tests="1">
							<testcase name="prefix some test name" classname="some test name">
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())
			Expect(parseResult.TestResults.Tests[0].Name).To(Equal("prefix some test name"))
		})

		It("calculates the correct name when the name is the same as classname", func() {
			parseResult, err := parsing.JUnitParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite tests="1">
							<testcase name="prefix some test name" classname="prefix some test name">
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())
			Expect(parseResult.TestResults.Tests[0].Name).To(Equal("prefix some test name"))
		})

		It("calculates the correct name when the name is entirely different from the classname", func() {
			parseResult, err := parsing.JUnitParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite tests="1">
							<testcase name="some test name" classname="prefix">
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())
			Expect(parseResult.TestResults.Tests[0].Name).To(Equal("prefix some test name"))
		})
	})
})