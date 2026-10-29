package performancereviewresolver

import (
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/workforce/app/performancereviewservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func reviewTemplateFromInput(
	input *gqlmodel.PerformanceReviewTemplateInput,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) *workerreview.PerformanceReviewTemplate {
	items := make([]workerreview.ReviewItem, 0, len(input.Items))
	for _, item := range input.Items {
		if item == nil {
			continue
		}
		items = append(items, workerreview.ReviewItem{
			Key:         strings.TrimSpace(item.Key),
			Label:       strings.TrimSpace(item.Label),
			Description: strings.TrimSpace(gqlscalar.StringValue(item.Description)),
			Weight:      int32(item.Weight), //nolint:gosec // bounded by validation
		})
	}
	entity := &workerreview.PerformanceReviewTemplate{
		ID:             id,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Code:           strings.TrimSpace(input.Code),
		Name:           strings.TrimSpace(input.Name),
		Description:    strings.TrimSpace(gqlscalar.StringValue(input.Description)),
		Status:         input.Status,
		IsDefault:      input.IsDefault,
		Items:          items,
		Version:        int64(gqlscalar.IntValue(input.Version)),
	}
	if input.CadenceMonths != nil {
		months := int32(*input.CadenceMonths) //nolint:gosec // bounded by validation
		entity.CadenceMonths = &months
	}
	return entity
}

func updateReviewRequestFromInput(
	input *gqlmodel.UpdatePerformanceReviewInput,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*performancereviewservice.UpdateReviewRequest, error) {
	id, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, errortypes.NewValidationError("id", errortypes.ErrInvalid, "Review is invalid")
	}
	ratings := make([]workerreview.ReviewRating, 0, len(input.Ratings))
	for _, rating := range input.Ratings {
		if rating == nil {
			continue
		}
		entry := workerreview.ReviewRating{
			Key:     rating.Key,
			Comment: gqlscalar.StringValue(rating.Comment),
		}
		if rating.Score != nil {
			score := int32(*rating.Score) //nolint:gosec // bounded by validation
			entry.Score = &score
		}
		ratings = append(ratings, entry)
	}
	goals := make([]workerreview.ReviewGoal, 0, len(input.Goals))
	for _, goal := range input.Goals {
		if goal == nil {
			continue
		}
		entry := workerreview.ReviewGoal{
			ID:    gqlscalar.StringValue(goal.ID),
			Title: goal.Title,
			DueAt: gqlscalar.Int64Ptr(goal.DueAt),
		}
		if goal.Status != nil {
			entry.Status = *goal.Status
		}
		goals = append(goals, entry)
	}
	return &performancereviewservice.UpdateReviewRequest{
		TenantInfo:   tenantInfo,
		ID:           id,
		Title:        gqlscalar.StringValue(input.Title),
		PeriodStart:  int64(gqlscalar.IntValue(input.PeriodStart)),
		PeriodEnd:    int64(gqlscalar.IntValue(input.PeriodEnd)),
		Ratings:      ratings,
		Summary:      gqlscalar.StringValue(input.Summary),
		Strengths:    gqlscalar.StringValue(input.Strengths),
		Improvements: gqlscalar.StringValue(input.Improvements),
		Goals:        goals,
		Version:      int64(input.Version),
		UserID:       userID,
	}, nil
}

func reviewStatusRequestFromInput(
	input *gqlmodel.PerformanceReviewStatusInput,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*performancereviewservice.ReviewStatusRequest, error) {
	id, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, errortypes.NewValidationError("id", errortypes.ErrInvalid, "Review is invalid")
	}
	return &performancereviewservice.ReviewStatusRequest{
		ID:         id,
		TenantInfo: tenantInfo,
		Version:    int64(gqlscalar.IntValue(input.Version)),
		UserID:     userID,
	}, nil
}

func reviewTemplateCursorConnectionToModel(
	result *pagination.CursorListResult[*workerreview.PerformanceReviewTemplate],
) (*gqlmodel.PerformanceReviewTemplateConnection, error) {
	edges, err := gqlconnection.EntityCursorEdges(
		result.Items,
		result,
		func(node *workerreview.PerformanceReviewTemplate, cursor string) *gqlmodel.PerformanceReviewTemplateEdge {
			return &gqlmodel.PerformanceReviewTemplateEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.PerformanceReviewTemplateConnection{
		Edges: edges,
		PageInfo: gqlconnection.PageInfo(
			result.HasNextPage,
			gqlconnection.LastEdgeCursor(
				edges,
				func(edge *gqlmodel.PerformanceReviewTemplateEdge) string { return edge.Cursor },
			),
		),
		TotalCount:      result.TotalCount,
		TotalCountExact: result.TotalCountExact,
	}, nil
}
