package formulastudioservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const (
	maxSaysRunes       = 400
	maxStatementErrors = 200
	explainMaxTokens   = 2500
)

const explainSystem = "You are the formula assistant in Trenova's Formula Studio. You explain a " +
	"rating formula to a billing analyst, one statement at a time. For every statement listed " +
	"under Engine trace, write one plain sentence of at most 30 words saying what that " +
	"statement works out and what it means for the charge. Write identifiers in backticks. " +
	amountsRule + " " + untrustedNote

func (s *Service) ExplainFormula(
	ctx context.Context,
	req *agentports.ExplainFormulaRequest,
	actor *permissionports.RequestActor,
) (*agentports.ExplainFormulaResponse, error) {
	definition, err := s.assistant(ctx, req.TenantInfo, actor)
	if err != nil {
		return nil, err
	}

	test := testRequest(&req.Draft, req.TenantInfo, req.SampleValues)
	test.TraceStatements = true
	if req.ShipmentID.IsNotNil() {
		shipmentID := req.ShipmentID
		test.ShipmentID = &shipmentID
	}
	evaluation, err := s.engine.TestExpression(ctx, test)
	if err != nil {
		return nil, err
	}
	if len(evaluation.Statements) == 0 {
		if failure := evaluation.Failure(); failure != nil {
			return nil, failure.ValidationError("expression", errortypes.ErrInvalid)
		}
		return nil, errortypes.NewValidationError(
			"expression",
			errortypes.ErrInvalid,
			"The expression must parse before it can be explained",
		)
	}

	resp := &agentports.ExplainFormulaResponse{
		Statements: evaluation.Statements,
		Guardrail:  evaluation.Guardrail,
	}
	if evaluation.Valid && evaluation.Result != nil {
		resp.BilledAmount = decimalPointer(*evaluation.Result)
		resp.RawAmount = decimalPointer(evaluation.RawAmount())
	}

	var reply explainReply
	err = s.complete(ctx, &completion{
		tenant:     req.TenantInfo,
		definition: definition,
		feature:    aiusage.FeatureFormulaExplain,
		system:     explainSystem,
		sections: append(draftSections(&req.Draft), agentports.ContextSection{
			Title:   "Engine trace",
			Content: traceText(i18n.FromContext(ctx), evaluation, resp),
		}),
		schema:     explainSchema(),
		schemaName: "formula_explanation",
		maxTokens:  explainMaxTokens,
	}, &reply)
	if err != nil {
		return nil, modelFailure(definition, err)
	}

	resp.Items = boundExplainItems(i18n.FromContext(ctx), reply.Items, evaluation.Statements)
	return resp, nil
}

type explainReply struct {
	Items []agentports.FormulaStatementNote `json:"items"`
}

func explainSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"items": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"statementIndex": map[string]any{
							"type":        "integer",
							"description": "The statement's index from Engine trace",
						},
						"says": map[string]any{
							"type":        "string",
							"description": "One sentence of at most 30 words",
						},
					},
					"required":             []string{"statementIndex", "says"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"items"},
		"additionalProperties": false,
	}
}

func traceText(
	locale i18n.Locale,
	evaluation *formulatemplateservice.TestExpressionResponse,
	resp *agentports.ExplainFormulaResponse,
) string {
	var builder strings.Builder
	for _, statement := range evaluation.Statements {
		builder.WriteString(fmt.Sprintf("[%d] ", statement.Index))
		if statement.Kind == formulatemplatetypes.StatementLet {
			builder.WriteString("let " + statement.Name + ": ")
		} else {
			builder.WriteString("result: ")
		}
		builder.WriteString(stringutils.OneLine(statement.Code, 2*maxRenderedValueRunes))
		if statement.Error != nil {
			builder.WriteString(" => fails (" + statement.Error.Code + "): ")
			builder.WriteString(stringutils.OneLine(
				statement.Error.Localize(locale),
				maxStatementErrors,
			))
		} else {
			builder.WriteString(" => ")
			builder.WriteString(renderValue(statement.Value))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("Formula before the charge policy: ")
	builder.WriteString(optionalText(resp.RawAmount))
	builder.WriteString("\nBilled after the charge policy: ")
	builder.WriteString(optionalText(resp.BilledAmount))
	if resp.Guardrail != nil && resp.Guardrail.Applied {
		builder.WriteString("\nThe ")
		builder.WriteString(boundName(resp.Guardrail.Bound))
		builder.WriteString(" replaced the formula's amount")
	}
	if !evaluation.Valid && evaluation.Error != nil {
		builder.WriteString(
			"\nThe formula fails on this load (" + evaluation.Error.Err.Code + "): ",
		)
		builder.WriteString(stringutils.OneLine(
			evaluation.Error.Err.Localize(locale),
			maxStatementErrors,
		))
	}

	return builder.String()
}

func boundExplainItems(
	locale i18n.Locale,
	items []agentports.FormulaStatementNote,
	statements []formulatemplatetypes.Statement,
) []agentports.FormulaStatementNote {
	says := make(map[int]string, len(statements))
	for _, item := range items {
		if item.StatementIndex < 0 || item.StatementIndex >= len(statements) {
			continue
		}
		if _, taken := says[item.StatementIndex]; taken {
			continue
		}
		text := stringutils.CollapseWhitespace(item.Says)
		if text == "" {
			continue
		}
		says[item.StatementIndex] = stringutils.Ellipsize(text, maxSaysRunes)
	}

	bounded := make([]agentports.FormulaStatementNote, 0, len(statements))
	for index := range statements {
		text, ok := says[index]
		if !ok {
			text = fallbackSays(locale, &statements[index])
		}
		bounded = append(bounded, agentports.FormulaStatementNote{
			StatementIndex: index,
			Says:           text,
		})
	}

	return bounded
}

func fallbackSays(locale i18n.Locale, statement *formulatemplatetypes.Statement) string {
	if statement.Kind == formulatemplatetypes.StatementLet {
		if statement.Error != nil {
			return fmt.Sprintf("`%s` fails on this load: %s", statement.Name,
				stringutils.OneLine(statement.Error.Localize(locale), maxStatementErrors))
		}
		return fmt.Sprintf("`%s` comes to %s on this load.", statement.Name,
			renderValue(statement.Value))
	}

	if statement.Error != nil {
		return "The formula fails on this load: " +
			stringutils.OneLine(statement.Error.Localize(locale), maxStatementErrors)
	}
	if amount := formulatemplateservice.TraceValueText(statement.Value); amount != nil {
		if parsed, err := decimal.NewFromString(*amount); err == nil {
			return fmt.Sprintf(
				"The formula comes to %s before the charge policy.",
				money.FormatDollars(parsed),
			)
		}
	}

	return "This is the amount the formula charges before the charge policy."
}

func boundName(bound string) string {
	if bound == "max" {
		return "maximum"
	}

	return "minimum"
}

func decimalPointer(value decimal.Decimal) *string {
	text := value.String()
	return &text
}
