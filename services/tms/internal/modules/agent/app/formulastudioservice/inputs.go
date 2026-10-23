package formulastudioservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxInputDescriptionRunes = 120
	describeMaxTokens        = 1500
)

const describeSystem = "You are the formula assistant in Trenova's Formula Studio. Write a doc " +
	"comment for each input listed under Inputs to describe: what the value is for, in words " +
	"a billing clerk understands, at most 120 characters, no trailing period needed. Read the " +
	"source to see how each input is used. Describe only the inputs listed there, by their " +
	"exact names. " + amountsRule + " " + untrustedNote

func (s *Service) DescribeInputs(
	ctx context.Context,
	req *agentports.DescribeFormulaInputsRequest,
	actor *permissionports.RequestActor,
) (*agentports.DescribeFormulaInputsResponse, error) {
	definition, err := s.assistant(ctx, req.TenantInfo, actor)
	if err != nil {
		return nil, err
	}

	tested, err := s.engine.TestSource(ctx, &formulatemplateservice.TestSourceRequest{
		TenantInfo: req.TenantInfo,
		Source:     req.Draft.Source,
	})
	if err != nil {
		return nil, err
	}

	blank := make(map[string]struct{}, len(tested.Inputs))
	names := make([]string, 0, len(tested.Inputs))
	for i := range tested.Inputs {
		input := &tested.Inputs[i]
		if strings.TrimSpace(input.Doc) != "" {
			continue
		}
		blank[input.Name] = struct{}{}
		names = append(names, input.Name)
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
		sections: append(draftSections(&req.Draft, tested.Inputs), agentports.ContextSection{
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
	return objectSchema(map[string]any{
		"descriptions": arraySchema(objectSchema(map[string]any{
			"name":        typedField("string", "The input's exact name"),
			"description": typedField("string", "At most 120 characters"),
		}, "name", "description"), ""),
	}, "descriptions")
}
