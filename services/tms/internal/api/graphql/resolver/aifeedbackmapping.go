package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/aifeedback"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func aiFeedbackColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AIFeedbackSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func aiFeedbackConnectionToModel(
	result *pagination.CursorListResult[*aifeedback.Feedback],
) (*gqlmodel.AIFeedbackConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *aifeedback.Feedback, cursor string) *gqlmodel.AIFeedbackEdge {
			return &gqlmodel.AIFeedbackEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AIFeedbackEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AIFeedbackConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func aiFeedbackTarget(
	targetType aifeedback.TargetType,
	targetID string,
	targetPart *string,
) (repositories.AIFeedbackTargetRef, error) {
	id, err := pulid.MustParse(targetID)
	if err != nil {
		return repositories.AIFeedbackTargetRef{}, errortypes.NewValidationError(
			"targetId",
			errortypes.ErrInvalid,
			"Target is not a valid id",
		)
	}

	return repositories.AIFeedbackTargetRef{
		TargetType: targetType,
		TargetID:   id,
		TargetPart: derefString(targetPart),
	}, nil
}

func aiFeedbackTargets(
	inputs []*gqlmodel.AIFeedbackTargetInput,
) ([]repositories.AIFeedbackTargetRef, error) {
	if len(inputs) > repositories.MaxAIFeedbackTargetsPerRead {
		return nil, errortypes.NewValidationError(
			"targets",
			errortypes.ErrInvalid,
			"Ask for at most {0} targets at a time",
			repositories.MaxAIFeedbackTargetsPerRead,
		)
	}

	targets := make([]repositories.AIFeedbackTargetRef, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		target, err := aiFeedbackTarget(input.TargetType, input.TargetID, input.TargetPart)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}

	return targets, nil
}

func aiFeedbackRating(value int) aifeedback.Rating {
	switch value {
	case int(aifeedback.RatingPositive):
		return aifeedback.RatingPositive
	case int(aifeedback.RatingNegative):
		return aifeedback.RatingNegative
	default:
		return 0
	}
}
