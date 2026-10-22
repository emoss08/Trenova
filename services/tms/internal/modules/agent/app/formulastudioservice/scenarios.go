package formulastudioservice

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	minSuggestedScenarios    = 3
	maxSuggestedScenarios    = 5
	maxScenarioNameRunes     = 100
	maxScenarioValues        = pagedraft.MaxFormulaVariables + 40
	maxExistingScenarios     = 50
	suggestScenariosMaxToken = 2000
)

const suggestSystem = "You are the formula assistant in Trenova's Formula Studio. Propose " +
	"three to five test scenarios for the formula: loads that pin its behaviour at the edges, " +
	"such as a minimum or maximum being reached, a break or threshold being crossed, an " +
	"optional value being empty, or a yes/no input being switched. Each has a short name, " +
	"one sentence saying what it checks, and the values that make it that load, set by the " +
	"names of the formula's inputs or the shipment fields its expression reads. Do not " +
	"repeat a scenario listed under Existing scenarios. Do not state amounts: the engine " +
	"prices every scenario. " + untrustedNote

func (s *Service) SuggestScenarios(
	ctx context.Context,
	req *agentports.SuggestFormulaScenariosRequest,
	actor *permissionports.RequestActor,
) (*agentports.SuggestFormulaScenariosResponse, error) {
	definition, err := s.assistant(ctx, req.TenantInfo, actor)
	if err != nil {
		return nil, err
	}

	var reply suggestReply
	err = s.complete(ctx, &completion{
		tenant:     req.TenantInfo,
		definition: definition,
		feature:    aiusage.FeatureFormulaGenerate,
		system:     suggestSystem,
		sections: append(draftSections(&req.Draft), agentports.ContextSection{
			Title:   "Existing scenarios",
			Content: existingText(req.Existing),
		}),
		schema:     suggestSchema(),
		schemaName: "formula_scenarios",
		maxTokens:  suggestScenariosMaxToken,
	}, &reply)
	if err != nil {
		return nil, modelFailure(definition, err)
	}

	suggestions := boundSuggestions(reply.Scenarios, req.Existing)
	if len(suggestions) == 0 {
		return nil, errortypes.NewBusinessError(
			"{0} did not suggest any scenarios that could be priced. Try again.",
			definition.Name,
		)
	}

	priced := make([]pagedraft.PricedScenario, 0, len(suggestions))
	for _, suggestion := range suggestions {
		price := s.engine.PriceScenario(ctx, &formulatemplateservice.PriceScenarioRequest{
			TenantInfo: req.TenantInfo,
			SchemaID:   schemaIDOf(&req.Draft),
			Content:    &req.Draft.Content,
			Inputs:     suggestion.Variables,
		})
		priced = append(priced, pagedraft.PricedScenario{
			Name:        suggestion.Name,
			Description: suggestion.Description,
			Variables:   suggestion.Variables,
			Amount:      price.Amount,
			Valid:       price.Valid,
			Error:       price.Error,
			Code:        price.Code,
		})
	}

	return &agentports.SuggestFormulaScenariosResponse{Scenarios: priced}, nil
}

type suggestedValue struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

type suggestedScenario struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Variables   []suggestedValue `json:"variables"`
}

type suggestReply struct {
	Scenarios []suggestedScenario `json:"scenarios"`
}

type boundSuggestion struct {
	Name        string
	Description string
	Variables   map[string]any
}

func boundSuggestions(
	scenarios []suggestedScenario,
	existing []agentports.ExistingFormulaScenario,
) []boundSuggestion {
	taken := make(map[string]struct{}, len(existing)+len(scenarios))
	for _, scenario := range existing {
		taken[strings.ToLower(strings.TrimSpace(scenario.Name))] = struct{}{}
	}

	bounded := make([]boundSuggestion, 0, maxSuggestedScenarios)
	for _, scenario := range scenarios {
		if len(bounded) == maxSuggestedScenarios {
			break
		}

		name := stringutils.CollapseWhitespace(scenario.Name)
		if name == "" || utf8.RuneCountInString(name) > maxScenarioNameRunes {
			continue
		}
		key := strings.ToLower(name)
		if _, dup := taken[key]; dup {
			continue
		}

		values, ok := boundValues(scenario.Variables)
		if !ok {
			continue
		}

		taken[key] = struct{}{}
		bounded = append(bounded, boundSuggestion{
			Name: name,
			Description: stringutils.Ellipsize(
				stringutils.CollapseWhitespace(scenario.Description),
				pagedraft.MaxVariableTextLength-1,
			),
			Variables: values,
		})
	}

	return bounded
}

func boundValues(values []suggestedValue) (map[string]any, bool) {
	if len(values) == 0 || len(values) > maxScenarioValues {
		return nil, false
	}

	bounded := make(map[string]any, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value.Name)
		if !pagedraft.ValidValuePath(name) ||
			!pagedraft.ScalarValue(value.Value, pagedraft.MaxVariableTextLength) {
			return nil, false
		}
		if value.Value == nil {
			continue
		}
		bounded[name] = value.Value
	}

	return bounded, len(bounded) > 0
}

func existingText(existing []agentports.ExistingFormulaScenario) string {
	if len(existing) == 0 {
		return "(none)"
	}

	limit := min(len(existing), maxExistingScenarios)
	lines := make([]string, 0, limit)
	for _, scenario := range existing[:limit] {
		lines = append(lines, fmt.Sprintf(
			"- %s: %s",
			stringutils.OneLine(scenario.Name, maxScenarioNameRunes),
			renderValues(scenario.Variables),
		))
	}

	return strings.Join(lines, "\n")
}

func suggestSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"scenarios": map[string]any{
				"type":     "array",
				"minItems": minSuggestedScenarios,
				"maxItems": maxSuggestedScenarios,
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type":        "string",
							"description": "A short name, such as \"Short haul under the minimum\"",
						},
						"description": map[string]any{
							"type":        "string",
							"description": "One sentence saying what the scenario checks",
						},
						"variables": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"name": map[string]any{
										"type": "string",
										"description": "An input's name or a shipment field " +
											"path such as totalDistance",
									},
									"value": map[string]any{
										"type": []string{"number", "string", "boolean"},
									},
								},
								"required":             []string{"name", "value"},
								"additionalProperties": false,
							},
						},
					},
					"required":             []string{"name", "description", "variables"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"scenarios"},
		"additionalProperties": false,
	}
}
