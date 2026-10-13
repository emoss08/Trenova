package formulastudioservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxInputDescriptionRunes = 120
	describeMaxTokens        = 1500
)

const describeSystem = "You are the formula assistant in Trenova's Formula Studio. Write a " +
	"description for each input listed under Inputs to describe: what the value is for, in " +
	"words a billing clerk understands, at most 120 characters, no trailing period needed. " +
	"Read the expression and the template to see how each input is used. Describe only the " +
	"inputs listed there, by their exact names. " + amountsRule + " " + untrustedNote

func (s *Service) DescribeInputs(
	ctx context.Context,
	req *agentports.DescribeFormulaInputsRequest,
	actor *permissionports.RequestActor,
) (*agentports.DescribeFormulaInputsResponse, error) {
	definition, err := s.assistant(ctx, req.TenantInfo, actor)
	if err != nil {
		return nil, err
	}

	blank := make(map[string]struct{}, len(req.Draft.Content.VariableDefinitions))
	names := make([]string, 0, len(req.Draft.Content.VariableDefinitions))
	for _, variable := range req.Draft.Content.VariableDefinitions {
		if variable == nil || variable.Name == "" ||
			strings.TrimSpace(variable.Description) != "" {
			continue
		}
		if _, dup := blank[variable.Name]; dup {
			continue
		}
		blank[variable.Name] = struct{}{}
		names = append(names, variable.Name)
	}

	resp := &agentports.DescribeFormulaInputsResponse{Descriptions: map[string]string{}}
	if len(names) == 0 {
		return resp, nil
	}

	var reply describeReply
	err = s.complete(ctx, &completion{
		tenant:     req.TenantInfo,
		definition: definition,
		feature:    aiusage.FeatureFormulaGenerate,
		system:     describeSystem,
		sections: append(draftSections(&req.Draft), agentports.ContextSection{
			Title:   "Inputs to describe",
			Content: strings.Join(names, "\n"),
		}),
		schema:     describeSchema(),
		schemaName: "formula_input_descriptions",
		maxTokens:  describeMaxTokens,
	}, &reply)
	if err != nil {
		return nil, modelFailure(definition, err)
	}

	for _, item := range reply.Descriptions {
		if _, wanted := blank[item.Name]; !wanted {
			continue
		}
		if _, done := resp.Descriptions[item.Name]; done {
			continue
		}
		text := stringutils.CollapseWhitespace(item.Description)
		if text == "" {
			continue
		}
		resp.Descriptions[item.Name] = stringutils.Ellipsize(text, maxInputDescriptionRunes-1)
	}

	return resp, nil
}

type describeReply struct {
	Descriptions []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"descriptions"`
}

func describeSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"descriptions": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type":        "string",
							"description": "The input's exact name",
						},
						"description": map[string]any{
							"type":        "string",
							"description": "At most 120 characters",
						},
					},
					"required":             []string{"name", "description"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"descriptions"},
		"additionalProperties": false,
	}
}
