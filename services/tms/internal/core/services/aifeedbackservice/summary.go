package aifeedbackservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aifeedback"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const secondsPerDay = int64(24 * 60 * 60)

func (s *Service) AgentSummary(
	ctx context.Context,
	req services.AgentFeedbackSummaryRequest,
) (*services.AgentFeedbackSummary, error) {
	if req.AgentDefinitionID.IsNil() {
		return nil, errortypes.NewValidationError(
			"agentDefinitionId",
			errortypes.ErrRequired,
			"Agent is required",
		)
	}

	window := req.WindowDays
	if window == 0 {
		window = services.DefaultAgentFeedbackWindowDays
	}
	if window < 1 || window > services.MaxAgentFeedbackWindowDays {
		return nil, errortypes.NewValidationError(
			"window",
			errortypes.ErrInvalid,
			"The window must be between 1 and {0} days",
			services.MaxAgentFeedbackWindowDays,
		)
	}

	worstLimit := req.WorstLimit
	if worstLimit <= 0 {
		worstLimit = services.DefaultAgentFeedbackWorstLimit
	}

	organization, err := s.organizations.GetByID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		return nil, err
	}

	since := s.now() - int64(window)*secondsPerDay
	windowReq := repositories.AIFeedbackWindowRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: req.AgentDefinitionID,
		Since:             since,
		Timezone:          timeutils.NormalizeTimezone(organization.Timezone),
		Limit:             worstLimit,
	}

	days, err := s.repo.DailySatisfaction(ctx, windowReq)
	if err != nil {
		return nil, err
	}

	scores, err := s.repo.WorstRated(ctx, windowReq)
	if err != nil {
		return nil, err
	}

	samples, err := s.samples(ctx, req, scores)
	if err != nil {
		return nil, err
	}

	summary := &services.AgentFeedbackSummary{
		AgentDefinitionID: req.AgentDefinitionID,
		WindowDays:        window,
		Since:             since,
		Days:              make([]*services.AgentFeedbackDay, 0, len(days)),
		WorstRated:        make([]*services.AgentFeedbackWorstRated, 0, len(scores)),
	}
	for _, day := range days {
		summary.Positive += day.Positive
		summary.Negative += day.Negative
		summary.Days = append(summary.Days, &services.AgentFeedbackDay{
			Day:          day.Day,
			Positive:     day.Positive,
			Negative:     day.Negative,
			Satisfaction: satisfaction(day.Positive, day.Negative),
		})
	}
	summary.Satisfaction = satisfaction(summary.Positive, summary.Negative)

	for _, score := range scores {
		summary.WorstRated = append(summary.WorstRated, &services.AgentFeedbackWorstRated{
			TargetType:  score.TargetType,
			TargetID:    score.TargetID,
			TargetPart:  score.TargetPart,
			Positive:    score.Positive,
			Negative:    score.Negative,
			LastRatedAt: score.LastRatedAt,
			Sample:      samples[score.SampleID],
		})
	}

	return summary, nil
}

func (s *Service) samples(
	ctx context.Context,
	req services.AgentFeedbackSummaryRequest,
	scores []*repositories.AIFeedbackTargetScore,
) (map[pulid.ID]*aifeedback.Feedback, error) {
	ids := make([]pulid.ID, 0, len(scores))
	for _, score := range scores {
		if score.SampleID.IsNotNil() {
			ids = append(ids, score.SampleID)
		}
	}
	if len(ids) == 0 {
		return map[pulid.ID]*aifeedback.Feedback{}, nil
	}

	rows, err := s.repo.ListByIDs(ctx, repositories.ListAIFeedbackByIDsRequest{
		TenantInfo: req.TenantInfo,
		IDs:        ids,
	})
	if err != nil {
		return nil, err
	}

	byID := make(map[pulid.ID]*aifeedback.Feedback, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}

	return byID, nil
}

func satisfaction(positive, negative int) *float64 {
	total := positive + negative
	if total == 0 {
		return nil
	}

	ratio := float64(positive) / float64(total)

	return &ratio
}
