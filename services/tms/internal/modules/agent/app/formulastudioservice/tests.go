package formulastudioservice

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/formulastudioports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatestudioservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxSuggestedTests    = 5
	maxTestNameRunes     = 100
	maxTestStatements    = 20
	maxTestStatementLen  = 400
	suggestTestsMaxToken = 2500
	explainTestMaxTokens = 1200
	maxWhyRunes          = 600
)

const suggestSystem = "You are the formula assistant in Trenova's Formula Studio. Propose up " +
	"to five Tariff test blocks for the formula that pin its behaviour at the edges: a minimum " +
	"or maximum being reached, a break or threshold being crossed, an optional field being " +
	"none, a condition switching a charge on or off. Each has a short name not used by the " +
	"source's existing tests and its statements: given lines that set shipment, lane, " +
	"rating or fuel fields or input.<name> values, then expect lines. Expect only what the " +
	"source makes true; for amounts, write expectations you can read from the source's " +
	"literal rates and the givens, and prefer comparisons such as charge(\"Label\") > $0 or " +
	"expect fails when an amount would need arithmetic you cannot be sure of. Statements " +
	"are single lines without the surrounding test braces. " + untrustedNote

func (s *Service) SuggestTests(
	ctx context.Context,
	req *formulastudioports.SuggestFormulaTestsRequest,
	actor *permissionports.RequestActor,
) (*formulastudioports.SuggestFormulaTestsResponse, error) {
	definition, err := s.assistant(ctx, req.TenantInfo, actor)
	if err != nil {
		return nil, err
	}

	current, err := s.studio.TestSource(ctx, &formulatemplatestudioservice.TestSourceRequest{
		TenantInfo: req.TenantInfo,
		Source:     req.Draft.Source,
	})
	if err != nil {
		return nil, err
	}
	locale := i18n.FromContext(ctx)
	if !current.Check.OK {
		return nil, errortypes.NewValidationError(
			"source",
			errortypes.ErrInvalid,
			"The source must check before tests can be suggested: {0}",
			diagnosticsText(locale, current.Check),
		)
	}

	taken := make(map[string]struct{}, len(current.Tests.Tests))
	for i := range current.Tests.Tests {
		taken[current.Tests.Tests[i].Name] = struct{}{}
	}

	var reply suggestReply
	err = s.complete(ctx, &completion{
		tenant:     req.TenantInfo,
		definition: definition,
		feature:    aiusage.FeatureFormulaGenerate,
		system:     suggestSystem,
		sections:   draftSections(&req.Draft, current.Inputs),
		schema:     suggestSchema(),
		schemaName: "formula_tests",
		maxTokens:  suggestTestsMaxToken,
	}, &reply)
	if err != nil {
		return nil, modelFailure(definition, err)
	}

	blocks := boundSuggestedTests(reply.Tests, taken)
	if len(blocks) == 0 {
		return nil, errortypes.NewBusinessError(
			"{0} did not suggest a test that could be run. Try again.",
			definition.Name,
		)
	}

	var source strings.Builder
	source.WriteString(strings.TrimRight(req.Draft.Source, "\n"))
	for _, block := range blocks {
		source.WriteString("\n\n" + block.Block)
	}
	source.WriteString("\n")

	run, err := s.studio.TestSource(ctx, &formulatemplatestudioservice.TestSourceRequest{
		TenantInfo: req.TenantInfo,
		Source:     source.String(),
	})
	if err != nil {
		return nil, err
	}

	results := make(map[string]*rules.TariffTestResult, len(run.Tests.Tests))
	for i := range run.Tests.Tests {
		results[run.Tests.Tests[i].Name] = &run.Tests.Tests[i]
	}
	for i := range blocks {
		result, ok := results[blocks[i].Name]
		if !ok {
			blocks[i].Status = rules.TariffTestError
			continue
		}
		blocks[i].Status = result.Status
		blocks[i].Failures = result.Failures
	}

	return &formulastudioports.SuggestFormulaTestsResponse{Tests: blocks}, nil
}

type suggestedTest struct {
	Name       string   `json:"name"`
	Statements []string `json:"statements"`
}

type suggestReply struct {
	Tests []suggestedTest `json:"tests"`
}

func suggestSchema() map[string]any {
	return objectSchema(map[string]any{
		"tests": arraySchema(objectSchema(map[string]any{
			"name": typedField("string", "The test's name, at most 100 characters"),
			"statements": arraySchema(
				typedField("string", ""),
				"given and expect lines, one per item",
			),
		}, "name", "statements"), ""),
	}, "tests")
}

func boundSuggestedTests(
	tests []suggestedTest,
	taken map[string]struct{},
) []formulastudioports.SuggestedFormulaTest {
	out := make([]formulastudioports.SuggestedFormulaTest, 0, maxSuggestedTests)
	for _, test := range tests {
		if len(out) >= maxSuggestedTests {
			break
		}
		name := stringutils.CollapseWhitespace(strings.ReplaceAll(test.Name, "\"", "'"))
		name = strings.NewReplacer("{", "(", "}", ")", "\\", "/").Replace(name)
		if name == "" || utf8.RuneCountInString(name) > maxTestNameRunes {
			continue
		}
		if _, dup := taken[name]; dup {
			continue
		}

		lines := make([]string, 0, len(test.Statements))
		for _, statement := range test.Statements {
			line := strings.TrimSpace(statement)
			if line == "" || strings.ContainsAny(line, "\n\r") ||
				strings.Count(line, "{") != strings.Count(line, "}") {
				continue
			}
			if len(line) > maxTestStatementLen ||
				!(strings.HasPrefix(line, "given ") || strings.HasPrefix(line, "expect ")) {
				continue
			}
			lines = append(lines, "  "+line)
			if len(lines) >= maxTestStatements {
				break
			}
		}
		if !hasExpectation(lines) {
			continue
		}

		taken[name] = struct{}{}
		out = append(out, formulastudioports.SuggestedFormulaTest{
			Name:  name,
			Block: fmt.Sprintf("test %q {\n%s\n}", name, strings.Join(lines, "\n")),
		})
	}
	return out
}

func hasExpectation(lines []string) bool {
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "expect ") {
			return true
		}
	}
	return false
}

const explainTestSystem = "You are the formula assistant in Trenova's Formula Studio. A test " +
	"block in the formula fails. From the source, the test and the engine's failure, say in at " +
	"most 80 words why it fails, and give a verdict: intended when the formula changed on " +
	"purpose and the test is out of date, bug when the formula is wrong, unclear when you " +
	"cannot tell. Write identifiers in backticks. " + amountsRule + " " + untrustedNote

func (s *Service) ExplainTest(
	ctx context.Context,
	req *formulastudioports.ExplainFormulaTestRequest,
	actor *permissionports.RequestActor,
) (*formulastudioports.ExplainFormulaTestResponse, error) {
	definition, err := s.assistant(ctx, req.TenantInfo, actor)
	if err != nil {
		return nil, err
	}

	run, err := s.studio.TestSource(ctx, &formulatemplatestudioservice.TestSourceRequest{
		TenantInfo: req.TenantInfo,
		Source:     req.Draft.Source,
	})
	if err != nil {
		return nil, err
	}
	locale := i18n.FromContext(ctx)

	var test *rules.TariffTestResult
	for i := range run.Tests.Tests {
		if run.Tests.Tests[i].Name == req.TestName {
			test = &run.Tests.Tests[i]
		}
	}
	if test == nil {
		return nil, errortypes.NewValidationError(
			"testName",
			errortypes.ErrNotFound,
			"The source has no test named {0}",
			req.TestName,
		)
	}

	resp := &formulastudioports.ExplainFormulaTestResponse{Test: test}
	if test.Status == rules.TariffTestPassed {
		resp.Verdict = formulastudioports.FormulaTestVerdictIntended
		resp.Why = "This test passes on the current source."
		return resp, nil
	}

	var reply explainTestReply
	err = s.complete(ctx, &completion{
		tenant:     req.TenantInfo,
		definition: definition,
		feature:    aiusage.FeatureFormulaExplain,
		system:     explainTestSystem,
		sections: append(draftSections(&req.Draft, run.Inputs), llmports.ContextSection{
			Title:   "Engine failure",
			Content: testFailureText(locale, test),
		}),
		schema:     explainTestSchema(),
		schemaName: "formula_test_explanation",
		maxTokens:  explainTestMaxTokens,
	}, &reply)
	switch {
	case err != nil && noProvider(err):
		resp.Verdict = formulastudioports.FormulaTestVerdictUnclear
		resp.Why = stringutils.Ellipsize(testFailureText(locale, test), maxWhyRunes)
		return resp, nil
	case err != nil:
		return nil, modelFailure(definition, err)
	}

	verdict := formulastudioports.FormulaTestVerdict(strings.TrimSpace(reply.Verdict))
	why := stringutils.CollapseWhitespace(reply.Why)
	if !verdict.IsValid() || why == "" {
		resp.Verdict = formulastudioports.FormulaTestVerdictUnclear
		resp.Why = stringutils.Ellipsize(testFailureText(locale, test), maxWhyRunes)
		return resp, nil
	}

	resp.Verdict = verdict
	resp.Why = stringutils.Ellipsize(why, maxWhyRunes)
	resp.Generated = true
	return resp, nil
}

type explainTestReply struct {
	Verdict string `json:"verdict"`
	Why     string `json:"why"`
}

func explainTestSchema() map[string]any {
	verdict := typedField("string", "")
	verdict["enum"] = []string{
		string(formulastudioports.FormulaTestVerdictIntended),
		string(formulastudioports.FormulaTestVerdictBug),
		string(formulastudioports.FormulaTestVerdictUnclear),
	}
	return objectSchema(map[string]any{
		"verdict": verdict,
		"why":     typedField("string", "Why the test fails, at most 80 words"),
	}, "verdict", "why")
}
