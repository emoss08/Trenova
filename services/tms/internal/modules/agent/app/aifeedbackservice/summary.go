package aifeedbackservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aifeedback"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgzone"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aifeedbackports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const secondsPerDay = int64(24 * 60 * 60)

func (s *Service) AgentSummary(
	ctx context.Context,
	req aifeedbackports.AgentFeedbackSummaryRequest,
) (*aifeedbackports.AgentFeedbackSummary, error) {
	if req.AgentDefinitionID.IsNil() {
		return nil, errortypes.NewValidationError(
			"agentDefinitionId",
			errortypes.ErrRequired,
			"Agent is required",
		)
	}

	window := req.WindowDays
	if window == 0 {
		window = aifeedbackports.DefaultAgentFeedbackWindowDays
	}
	if window < 1 || window > aifeedbackports.MaxAgentFeedbackWindowDays {
		return nil, errortypes.NewValidationError(
			"window",
			errortypes.ErrInvalid,
			"The window must be between 1 and {0} days",
			aifeedbackports.MaxAgentFeedbackWindowDays,
		)
	}

	worstLimit := req.WorstLimit
	if worstLimit <= 0 {
		worstLimit = aifeedbackports.DefaultAgentFeedbackWorstLimit
	}

	location, err := orgzone.Location(ctx, s.organizations, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	since := s.now() - int64(window)*secondsPerDay
	windowReq := aifeedbackports.AIFeedbackWindowRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: req.AgentDefinitionID,
		Since:             since,
		Timezone:          location.String(),
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

	summary := &aifeedbackports.AgentFeedbackSummary{
		AgentDefinitionID: req.AgentDefinitionID,
		WindowDays:        window,
		Since:             since,
		Days:              make([]*aifeedbackports.AgentFeedbackDay, 0, len(days)),
		WorstRated:        make([]*aifeedbackports.AgentFeedbackWorstRated, 0, len(scores)),
	}
	for _, day := range days {
		summary.Positive += day.Positive
		summary.Negative += day.Negative
		summary.Days = append(summary.Days, &aifeedbackports.AgentFeedbackDay{
			Day:          day.Day,
			Positive:     day.Positive,
			Negative:     day.Negative,
			Satisfaction: satisfaction(day.Positive, day.Negative),
		})
	}
	summary.Satisfaction = satisfaction(summary.Positive, summary.Negative)

	for _, score := range scores {
		summary.WorstRated = append(summary.WorstRated, &aifeedbackports.AgentFeedbackWorstRated{
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
	req aifeedbackports.AgentFeedbackSummaryRequest,
	scores []*aifeedbackports.AIFeedbackTargetScore,
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

	rows, err := s.repo.ListByIDs(ctx, aifeedbackports.ListAIFeedbackByIDsRequest{
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
