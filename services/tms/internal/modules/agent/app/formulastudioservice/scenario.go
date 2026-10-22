package formulastudioservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const (
	maxWhyRunes          = 320
	maxWhySentences      = 2
	explainScenarioToken = 600
)

const whySystem = "You are the formula assistant in Trenova's Formula Studio. A test scenario " +
	"on a formula template no longer bills what it expects. The formula engine evaluated the " +
	"content the expectation was recorded on and the editor's content on the scenario's " +
	"inputs; its numbers are authoritative, so do not recompute them. In at most two short " +
	"sentences of plain English for a pricing analyst, name the root cause of the difference, " +
	"writing the specific input or term in backticks. Format money as dollars like $1,234.56; " +
	"inputs that are not money stay plain numbers. Then judge whether this looks like an " +
	"intended rate change (the expectation should be updated: intended), a regression (the " +
	"formula should be fixed: bug), or neither is clear (unclear). " + amountsRule + " " +
	untrustedNote

func (s *Service) ExplainScenario(
	ctx context.Context,
	req *agentports.ExplainFormulaScenarioRequest,
	actor *permissionports.RequestActor,
) (*agentports.ExplainFormulaScenarioResponse, error) {
	definition, err := s.assistant(ctx, req.TenantInfo, actor)
	if err != nil {
		return nil, err
	}

	candidate := req.Candidate
	trace, err := s.engine.ScenarioTrace(ctx, &formulatemplateservice.ScenarioTraceRequest{
		TenantInfo: req.TenantInfo,
		TemplateID: req.TemplateID,
		TestCaseID: req.TestCaseID,
		Candidate:  &candidate,
	})
	if err != nil {
		return nil, err
	}

	resp := scenarioResponse(trace)

	var reply whyReply
	err = s.complete(ctx, &completion{
		tenant:     req.TenantInfo,
		definition: definition,
		feature:    aiusage.FeatureFormulaExplain,
		system:     whySystem,
		sections: []agentports.ContextSection{
			{Title: "Scenario", Content: scenarioText(trace)},
			{Title: "Engine trace", Content: scenarioTraceText(trace)},
			{Title: "Current expression", Content: trace.CandidateExpression},
		},
		schema:     whySchema(),
		schemaName: "scenario_cause",
		maxTokens:  explainScenarioToken,
	}, &reply)
	if err != nil {
		if !noProvider(err) {
			s.l.Warn("the scenario cause fell back to the engine's own reading", zap.Error(err))
		}
		return resp, nil
	}

	verdict := formulatemplateports.ScenarioVerdict(strings.TrimSpace(reply.Verdict))
	why := boundWhy(reply.Why)
	if !verdict.IsValid() || why == "" {
		return resp, nil
	}

	resp.Verdict = verdict
	resp.Why = why
	resp.Generated = true
	return resp, nil
}

func scenarioResponse(
	trace *formulatemplateports.ScenarioTrace,
) *agentports.ExplainFormulaScenarioResponse {
	rows := trace.Rows
	if rows == nil {
		rows = []formulatemplateports.ScenarioTraceRow{}
	}

	return &agentports.ExplainFormulaScenarioResponse{
		Verdict:           trace.FallbackVerdict,
		Why:               trace.FallbackWhy,
		Generated:         false,
		Expected:          trace.Expected,
		Actual:            trace.Actual,
		Delta:             trace.Delta,
		BaselineLabel:     trace.BaselineLabel,
		RecordedOnVersion: trace.RecordedOnVersion,
		RecordedBy:        trace.RecordedBy,
		Error:             trace.Error,
		ErrorCode:         trace.ErrorCode,
		Rows:              rows,
	}
}

type whyReply struct {
	Why     string `json:"why"`
	Verdict string `json:"verdict"`
}

func whySchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"why": map[string]any{
				"type":        "string",
				"description": "At most two short sentences",
			},
			"verdict": map[string]any{
				"type": "string",
				"enum": []string{
					string(formulatemplateports.ScenarioVerdictIntended),
					string(formulatemplateports.ScenarioVerdictBug),
					string(formulatemplateports.ScenarioVerdictUnclear),
				},
			},
		},
		"required":             []string{"why", "verdict"},
		"additionalProperties": false,
	}
}

func boundWhy(raw string) string {
	text := stringutils.CollapseWhitespace(raw)
	if text == "" {
		return ""
	}

	sentences := 0
	cut := len(text)
	for index, r := range text {
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		next := index + 1
		if next < len(text) && text[next] != ' ' {
			continue
		}
		sentences++
		if sentences == maxWhySentences {
			cut = next
			break
		}
	}

	return stringutils.Ellipsize(text[:cut], maxWhyRunes)
}

func scenarioText(trace *formulatemplateports.ScenarioTrace) string {
	recordedBy := ""
	if trace.RecordedBy != nil && trace.RecordedBy.Name != "" {
		recordedBy = " by " + trace.RecordedBy.Name
	}

	return fmt.Sprintf(
		"Name: %s\nInputs: %s\nExpected: %s, recorded on %s%s",
		stringutils.OneLine(trace.ScenarioName, maxScenarioNameRunes),
		renderValues(trace.ScenarioInputs),
		trace.Expected,
		trace.BaselineLabel,
		recordedBy,
	)
}

func scenarioTraceText(trace *formulatemplateports.ScenarioTrace) string {
	var builder strings.Builder
	if len(trace.Rows) == 0 {
		builder.WriteString("No differences between the recorded content and the editor\n")
	}
	for _, row := range trace.Rows {
		builder.WriteString(fmt.Sprintf(
			"%s %s: %s -> %s\n",
			row.Kind,
			row.Name,
			stringutils.OneLine(optionalText(row.Before), maxRenderedValueRunes),
			stringutils.OneLine(optionalText(row.After), maxRenderedValueRunes),
		))
	}

	builder.WriteString(fmt.Sprintf(
		"formula before the policy: %s -> %s\n",
		optionalText(trace.RecordedRaw),
		optionalText(trace.CandidateRaw),
	))
	builder.WriteString(fmt.Sprintf(
		"policy clamp: %s -> %s\n",
		stringutils.WithDefault(trace.RecordedClamp, "none"),
		stringutils.WithDefault(trace.CandidateClamp, "none"),
	))
	builder.WriteString(fmt.Sprintf(
		"billed: %s -> %s\n",
		optionalText(trace.RecordedBilled),
		optionalText(trace.Actual),
	))
	builder.WriteString(fmt.Sprintf(
		"expression changed: %t; policy changed: %t",
		trace.ExpressionChanged,
		trace.PolicyChanged,
	))
	if trace.Error != "" {
		builder.WriteString("\nruntime error: ")
		builder.WriteString(stringutils.OneLine(trace.Error, maxStatementErrors))
	}

	return builder.String()
}
