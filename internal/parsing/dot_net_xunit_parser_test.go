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

var _ = Describe("DotNetxUnitParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/xunit_dot_net.xml")
			Expect(err).ToNot(HaveOccurred())

			parseResult, err := parsing.DotNetxUnitParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())

			Expect(parseResult.Parser).To(Equal(parsing.DotNetxUnitParser{}))
			Expect(parseResult.Sentiment).To(Equal(parsing.PositiveParseResultSentiment))
			Expect(parseResult.TestResults.Framework.Language).To(Equal(v1.FrameworkLanguageDotNet))
			Expect(parseResult.TestResults.Framework.Kind).To(Equal(v1.FrameworkKindxUnit))
			Expect(parseResult.TestResults.Summary.Tests).To(Equal(15))
			Expect(parseResult.TestResults.Summary.Successful).To(Equal(13))
			Expect(parseResult.TestResults.Summary.Skipped).To(Equal(1))
			Expect(parseResult.TestResults.Summary.Failed).To(Equal(1))
			Expect(parseResult.TestResults.Summary.OtherErrors).To(Equal(0))
		})

		It("errors on malformed XML", func() {
			parseResult, err := parsing.DotNetxUnitParser{}.Parse(strings.NewReader(`<abc`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(parseResult).To(BeNil())
		})

		It("errors on XML that doesn't look like xUnit.NET", func() {
			var parseResult *parsing.ParseResult
			var err error

			parseResult, err = parsing.DotNetxUnitParser{}.Parse(strings.NewReader(`<foo></foo>`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(parseResult).To(BeNil())

			parseResult, err = parsing.DotNetxUnitParser{}.Parse(strings.NewReader(`<assemblies></assemblies>`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The test suites in the XML do not appear to match xUnit.NET XML"))
			Expect(parseResult).To(BeNil())

			parseResult, err = parsing.DotNetxUnitParser{}.Parse(
				strings.NewReader(`<assemblies><assembly></assembly></assemblies>`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("The test suites in the XML do not appear to match xUnit.NET XML"),
			)
			Expect(parseResult).To(BeNil())
		})

		It("can extract a detailed successful test", func() {
			parseResult, err := parsing.DotNetxUnitParser{}.Parse(strings.NewReader(
				`
					<assemblies>
						<assembly name="some/path/to/AssemblyName.dll">
							<collection>
								<test
									id="some-id"
									name="NullAssertsTests+Null.Success"
									type="NullAssertsTests+Null"
									method="Success"
									time="0.0063709"
									result="Pass"
									source-file="some/path/to/source.cs"
									source-line="12"
								>
									<traits>
										<trait name="some-trait" value="some-value" />
										<trait name="other-trait" value="other-value" />
									</traits>
									<output><![CDATA[line 1
line 2
line 3]]></output>
								</test>
							</collection>
						</assembly>
					</assemblies>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())

			line := 12
			id := "some-id"
			duration := time.Duration(6370900)
			testType := "NullAssertsTests+Null"
			testMethod := "Success"
			stdout := "line 1\nline 2\nline 3"
			Expect(parseResult.TestResults.Tests[0]).To(Equal(
				v1.Test{
					ID:       &id,
					Name:     "NullAssertsTests+Null.Success",
					Location: &v1.Location{File: "some/path/to/source.cs", Line: &line},
					Attempt: v1.TestAttempt{
						Duration: &duration,
						Meta: map[string]any{
							"assembly":          "AssemblyName.dll",
							"type":              &testType,
							"method":            &testMethod,
							"trait-some-trait":  "some-value",
							"trait-other-trait": "other-value",
						},
						Status: v1.NewSuccessfulTestStatus(),
						Stdout: &stdout,
					},
				},
			))
		})

		It("can extract a failed test", func() {
			parseResult, err := parsing.DotNetxUnitParser{}.Parse(strings.NewReader(
				`
					<assemblies>
						<assembly name="some/path/to/AssemblyName.dll">
							<collection>
								<test
									name="NullAssertsTests+Null.Success"
									time="0.0063709"
									result="Fail"
								>
									<failure exception-type="AssertionException">
										<message><![CDATA[Some message here]]></message>
										<stack-trace><![CDATA[Some trace
											other line]]></stack-trace>
									</failure>
								</test>
							</collection>
						</assembly>
					</assemblies>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())

			duration := time.Duration(6370900)
			message := "Some message here"
			exception := "AssertionException"
			var zeroString *string
			Expect(parseResult.TestResults.Tests[0]).To(Equal(
				v1.Test{
					Name: "NullAssertsTests+Null.Success",
					Attempt: v1.TestAttempt{
						Duration: &duration,
						Meta: map[string]any{
							"assembly": "AssemblyName.dll",
							"type":     zeroString,
							"method":   zeroString,
						},
						Status: v1.NewFailedTestStatus(&message, &exception, []string{"Some trace", "other line"}),
					},
				},
			))
		})

		It("can extract a failed test without details", func() {
			parseResult, err := parsing.DotNetxUnitParser{}.Parse(strings.NewReader(
				`
					<assemblies>
						<assembly name="some/path/to/AssemblyName.dll">
							<collection>
								<test
									name="NullAssertsTests+Null.Success"
									time="0.0063709"
									result="Fail"
								>
								</test>
							</collection>
						</assembly>
					</assemblies>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())

			duration := time.Duration(6370900)
			var zeroString *string
			Expect(parseResult.TestResults.Tests[0]).To(Equal(
				v1.Test{
					Name: "NullAssertsTests+Null.Success",
					Attempt: v1.TestAttempt{
						Duration: &duration,
						Meta: map[string]any{
							"assembly": "AssemblyName.dll",
							"type":     zeroString,
							"method":   zeroString,
						},
						Status: v1.NewFailedTestStatus(nil, nil, nil),
					},
				},
			))
		})

		It("can extract a skipped test", func() {
			parseResult, err := parsing.DotNetxUnitParser{}.Parse(strings.NewReader(
				`
					<assemblies>
						<assembly name="some/path/to/AssemblyName.dll">
							<collection>
								<test
									name="NullAssertsTests+Null.Success"
									time="0.0063709"
									result="Skip"
								>
									<reason><![CDATA[Some reason here]]></reason>
								</test>
							</collection>
						</assembly>
					</assemblies>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())

			duration := time.Duration(6370900)
			message := "Some reason here"
			var zeroString *string
			Expect(parseResult.TestResults.Tests[0]).To(Equal(
				v1.Test{
					Name: "NullAssertsTests+Null.Success",
					Attempt: v1.TestAttempt{
						Duration: &duration,
						Meta: map[string]any{
							"assembly": "AssemblyName.dll",
							"type":     zeroString,
							"method":   zeroString,
						},
						Status: v1.NewSkippedTestStatus(&message),
					},
				},
			))
		})

		It("can extract a not-run test", func() {
			parseResult, err := parsing.DotNetxUnitParser{}.Parse(strings.NewReader(
				`
					<assemblies>
						<assembly name="some/path/to/AssemblyName.dll">
							<collection>
								<test
									name="NullAssertsTests+Null.Success"
									time="0.0063709"
									result="NotRun"
								>
								</test>
							</collection>
						</assembly>
					</assemblies>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())

			duration := time.Duration(6370900)
			var zeroString *string
			Expect(parseResult.TestResults.Tests[0]).To(Equal(
				v1.Test{
					Name: "NullAssertsTests+Null.Success",
					Attempt: v1.TestAttempt{
						Duration: &duration,
						Meta: map[string]any{
							"assembly": "AssemblyName.dll",
							"type":     zeroString,
							"method":   zeroString,
						},
						Status: v1.NewSkippedTestStatus(nil),
					},
				},
			))
		})

		It("errors on other results", func() {
			parseResult, err := parsing.DotNetxUnitParser{}.Parse(strings.NewReader(
				`
					<assemblies>
						<assembly name="some/path/to/AssemblyName.dll">
							<collection>
								<test
									name="NullAssertsTests+Null.Success"
									time="0.0063709"
									result="wat"
								>
								</test>
							</collection>
						</assembly>
					</assemblies>
				`,
			))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`Unexpected result "wat"`))
			Expect(parseResult).To(BeNil())
		})

		It("can extract other errors", func() {
			parseResult, err := parsing.DotNetxUnitParser{}.Parse(strings.NewReader(
				`
					<assemblies>
						<assembly name="some/path/to/AssemblyName.dll">
							<errors>
								<error name="ErrorName" type="error-type-one">
									<failure exception-type="SomeException">
										<message><![CDATA[Some message here]]></message>
										<stack-trace><![CDATA[Some trace
											other line]]></stack-trace>
									</failure>
								</error>
								<error type="error-type-two">
									<failure />
								</error>
							</errors>
							<collection>
							</collection>
						</assembly>
					</assemblies>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())
			exception := "SomeException"
			Expect(parseResult.TestResults.OtherErrors).To(Equal(
				[]v1.OtherError{
					{
						Backtrace: []string{"Some trace", "other line"},
						Exception: &exception,
						Message:   "Some message here",
						Meta: map[string]any{
							"assembly": "AssemblyName.dll",
							"type":     "error-type-one",
						},
					},
					{
						Message: "An error occurred during error-type-two",
						Meta: map[string]any{
							"assembly": "AssemblyName.dll",
							"type":     "error-type-two",
						},
					},
				},
			))
		})
	})
})