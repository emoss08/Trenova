package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

func agentEvalCaseColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AgentEvalCaseSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func agentEvalCaseConnectionToModel(
	result *pagination.CursorListResult[*agentquality.EvalCase],
) (*gqlmodel.AgentEvalCaseConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *agentquality.EvalCase, cursor string) *gqlmodel.AgentEvalCaseEdge {
			return &gqlmodel.AgentEvalCaseEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AgentEvalCaseEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AgentEvalCaseConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func jsonList[T any](items []T) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for i := range items {
		out = append(out, jsonutils.MustToJSON(&items[i]))
	}

	return out
}

func evalCaseSources(input gqlmodel.CreateAgentEvalCaseInput) int {
	count := 0
	for _, set := range []bool{
		input.FromMessage != nil,
		input.FromProposal != nil,
		input.Curated != nil,
	} {
		if set {
			count++
		}
	}

	return count
}

func evalCaseFromMessage(
	input *gqlmodel.AgentEvalCaseFromMessageInput,
) (*services.CreateEvalCaseFromMessageRequest, error) {
	threadID, err := pulid.MustParse(input.ThreadID)
	if err != nil {
		return nil, err
	}
	messageID, err := pulid.MustParse(input.MessageID)
	if err != nil {
		return nil, err
	}

	request := &services.CreateEvalCaseFromMessageRequest{
		ThreadID:  threadID,
		MessageID: messageID,
		Title:     derefString(input.Title),
	}
	if input.FeedbackID != nil && *input.FeedbackID != "" {
		if request.FeedbackID, err = pulid.MustParse(*input.FeedbackID); err != nil {
			return nil, err
		}
	}

	return request, nil
}

func evalCaseCurated(
	input *gqlmodel.AgentEvalCaseCuratedInput,
) (*services.CreateCuratedEvalCaseRequest, error) {
	definitionID, err := pulid.MustParse(input.AgentDefinitionID)
	if err != nil {
		return nil, err
	}

	expected, err := decodeExpected(input.Expected)
	if err != nil {
		return nil, err
	}

	request := &services.CreateCuratedEvalCaseRequest{
		AgentDefinitionID: definitionID,
		Title:             derefString(input.Title),
		Input:             input.Input,
		HeldTools:         input.HeldTools,
		Expected:          *expected,
		Rubric:            derefString(input.Rubric),
		ExpiresAt:         optionalInt64(input.ExpiresAt),
	}
	if input.Trigger != nil {
		request.Trigger = *input.Trigger
	}
	if input.PageContext != nil {
		request.PageContext = new(agent.PageContext)
		if err = jsonutils.Convert(input.PageContext, request.PageContext); err != nil {
			return nil, invalidJSON(
				"pageContext",
				"The page context is not in a shape the agent reads",
			)
		}
	}
	if len(input.Mentions) > 0 {
		if err = jsonutils.Convert(input.Mentions, &request.Mentions); err != nil {
			return nil, invalidJSON(
				"mentions",
				"Mentions must be records with a type, id and label",
			)
		}
		request.Mentions = agent.NormalizeEntityRefs(request.Mentions)
	}
	if input.SubjectType != nil && *input.SubjectType != "" {
		request.SubjectType = agent.SubjectType(*input.SubjectType)
	}
	if input.SubjectID != nil && *input.SubjectID != "" {
		if request.SubjectID, err = pulid.MustParse(*input.SubjectID); err != nil {
			return nil, err
		}
	}

	return request, nil
}

func evalCaseUpdate(
	caseID pulid.ID,
	input gqlmodel.UpdateAgentEvalCaseInput,
) (*services.UpdateEvalCaseRequest, error) {
	request := &services.UpdateEvalCaseRequest{
		ID:        caseID,
		Version:   int64(input.Version),
		Title:     input.Title,
		Input:     input.Input,
		HeldTools: input.HeldTools,
		Rubric:    input.Rubric,
	}
	if input.Expected != nil {
		expected, err := decodeExpected(input.Expected)
		if err != nil {
			return nil, err
		}
		request.Expected = expected
	}
	if input.ExpiresAt.IsSet() {
		value := input.ExpiresAt.Value()
		if value == nil {
			request.ClearExpiry = true
		} else {
			request.ExpiresAt = optionalInt64(value)
		}
	}

	return request, nil
}

func decodeExpected(raw map[string]any) (*agentquality.Expected, error) {
	expected := new(agentquality.Expected)
	if err := jsonutils.Convert(raw, expected); err != nil {
		return nil, invalidJSON(
			"expected",
			"What the case expects is not in a shape the scorer reads",
		)
	}

	return expected, nil
}

func invalidJSON(field, message string) error {
	return errortypes.NewValidationError(field, errortypes.ErrInvalid, message)
}
