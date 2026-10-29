package agentqualityservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitysuite"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aifeedback"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aifeedbackports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	secondsPerDay         = int64(24 * 60 * 60)
	agentsCursorScope     = "agent-quality-agents"
	worstRatedCursorScope = "agent-worst-rated"
	detailWorstRatedLimit = 5
)

func windowDays(requested int) (int, error) {
	if requested == 0 {
		return agentqualityports.DefaultAgentQualityWindowDays, nil
	}
	if requested < 1 || requested > agentqualityports.MaxAgentQualityWindowDays {
		return 0, errortypes.NewValidationError(
			"window",
			errortypes.ErrInvalid,
			"The window must be between 1 and {0} days",
			agentqualityports.MaxAgentQualityWindowDays,
		)
	}

	return requested, nil
}

func satisfaction(positive, negative int) *float64 {
	total := positive + negative
	if total == 0 {
		return nil
	}
	ratio := float64(positive) / float64(total)

	return &ratio
}

func (s *Service) Overview(
	ctx context.Context,
	req *agentqualityports.AgentQualityOverviewRequest,
) (*agentqualityports.AgentQualityOverview, error) {
	window, err := windowDays(req.WindowDays)
	if err != nil {
		return nil, err
	}

	control, err := s.control(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	now := s.now()
	since := now - int64(window)*secondsPerDay
	timezone := s.timezoneOf(ctx, control, req.TenantInfo)
	_, monthStart, err := windows(now, timezone)
	if err != nil {
		return nil, err
	}

	overview := &agentqualityports.AgentQualityOverview{
		WindowDays:          window,
		Since:               since,
		RatingsVisible:      req.IncludeRatings,
		MonthlyBudgetUSD:    control.MonthlyBudgetUSD.StringFixed(2),
		MonthStartedAt:      monthStart,
		SweepEnabled:        control.Enabled,
		NextSweepHourLocal:  control.RunHourLocal,
		NextSweepTimezone:   timezone,
		JudgeEnabled:        control.JudgeEnabled,
		RegressionThreshold: control.RegressionThreshold,
	}

	if req.IncludeRatings {
		totals, totalsErr := s.feedback.TotalsByAgent(
			ctx,
			aifeedbackports.AIFeedbackAgentTotalsRequest{
				TenantInfo: req.TenantInfo,
				Since:      since,
			},
		)
		if totalsErr != nil {
			return nil, totalsErr
		}
		positive, negative := 0, 0
		for _, total := range totals {
			positive += total.Positive
			negative += total.Negative
		}
		overview.Ratings = positive + negative
		overview.Satisfaction = satisfaction(positive, negative)
	}

	runs, err := s.suiteRuns.Overview(ctx, agentqualityports.AgentSuiteRunOverviewRequest{
		TenantInfo: req.TenantInfo,
		Since:      since,
	})
	if err != nil {
		return nil, err
	}
	overview.SuiteRuns = runs.Runs
	overview.Regressions = runs.Regressions

	latest, err := s.suiteRuns.Latest(ctx, agentqualityports.LatestAgentSuiteRunsRequest{
		TenantInfo: req.TenantInfo,
		Statuses:   scoredStatuses(),
	})
	if err != nil {
		return nil, err
	}
	scores := make([]float64, 0, len(latest))
	for _, run := range latest {
		if run.QualityScore != nil {
			scores = append(scores, *run.QualityScore)
		}
		if run.Regression {
			overview.OpenRegressions++
		}
	}
	overview.WorstRegression = worstRegression(latest)
	overview.AgentsScored = len(scores)
	overview.QualityScore = meanPointer(scores)

	withCases, err := s.cases.ListAgentsWithActiveCases(
		ctx,
		agentqualityports.ListAgentsWithActiveEvalCasesRequest{TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}
	overview.AgentsWithCases = len(withCases)

	spend, err := s.usage.EvaluationCost(ctx, aiusageports.AIUsageEvaluationCostRequest{
		TenantInfo: req.TenantInfo,
		Since:      monthStart,
	})
	if err != nil {
		return nil, err
	}
	overview.EvalSpendMonthUSD = spend.CostUSD.StringFixed(2)
	overview.EvalUnpricedCalls = spend.UnpricedCalls

	return overview, nil
}

func worstRegression(latest []*agentqualitysuite.SuiteRun) *agentqualitysuite.SuiteRun {
	var worst *agentqualitysuite.SuiteRun
	for _, run := range latest {
		if run == nil || !run.Regression {
			continue
		}
		if worst == nil || regressionDrop(run) > regressionDrop(worst) {
			worst = run
		}
	}

	return worst
}

func regressionDrop(run *agentqualitysuite.SuiteRun) float64 {
	if run == nil || run.QualityScore == nil || run.BaselineScore == nil {
		return 0
	}

	return *run.BaselineScore - *run.QualityScore
}

func qualityPoints(runs []*agentqualitysuite.SuiteRun) []*agentqualityports.AgentQualityPoint {
	points := make([]*agentqualityports.AgentQualityPoint, 0, len(runs))
	for _, run := range runs {
		if run.QualityScore == nil {
			continue
		}
		at := run.StartedAt
		if run.FinishedAt != nil {
			at = *run.FinishedAt
		}
		points = append(points, &agentqualityports.AgentQualityPoint{
			SuiteRunID:   run.ID,
			At:           at,
			QualityScore: *run.QualityScore,
			Status:       run.Status,
			Regression:   run.Regression,
		})
	}

	return points
}

func (s *Service) AgentQuality(
	ctx context.Context,
	req *agentqualityports.AgentQualityRequest,
) (*agentqualityports.AgentQualityDetail, error) {
	window, err := windowDays(req.WindowDays)
	if err != nil {
		return nil, err
	}

	definition, err := s.definitions.GetByID(
		ctx,
		agentdefinitionports.GetAgentDefinitionByIDRequest{
			ID:         req.AgentDefinitionID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	now := s.now()
	since := now - int64(window)*secondsPerDay
	detail := &agentqualityports.AgentQualityDetail{
		AgentDefinitionID:  definition.ID,
		AgentName:          definition.Name,
		Enabled:            definition.Enabled,
		WindowDays:         window,
		Since:              since,
		RatingsVisible:     req.IncludeRatings,
		SatisfactionPoints: []*aifeedbackports.AgentFeedbackDay{},
		WorstRated:         []*agentqualityports.AgentWorstRatedAnswer{},
	}

	if req.IncludeRatings {
		if err = s.fillRatings(ctx, detail, req, since); err != nil {
			return nil, err
		}
	}

	history, err := s.suiteRuns.History(ctx, agentqualityports.AgentSuiteRunHistoryRequest{
		TenantInfo:         req.TenantInfo,
		AgentDefinitionIDs: []pulid.ID{definition.ID},
		Since:              since,
		PerAgent:           agentqualityports.AgentQualityHistoryPoints,
	})
	if err != nil {
		return nil, err
	}
	detail.QualityPoints = qualityPoints(history)

	last, err := s.suiteRuns.Last(ctx, agentqualityports.LastAgentSuiteRunRequest{
		AgentDefinitionID: definition.ID,
		TenantInfo:        req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	detail.LastSuiteRun = last

	cases, err := s.cases.ListSamplingCases(ctx, agentqualityports.ListEvalCaseSamplingRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: definition.ID,
	})
	if err != nil {
		return nil, err
	}
	detail.ActiveCases = len(cases)

	return detail, nil
}

func (s *Service) fillRatings(
	ctx context.Context,
	detail *agentqualityports.AgentQualityDetail,
	req *agentqualityports.AgentQualityRequest,
	since int64,
) error {
	control, err := s.control(ctx, req.TenantInfo)
	if err != nil {
		return err
	}

	windowReq := aifeedbackports.AIFeedbackWindowRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: req.AgentDefinitionID,
		Since:             since,
		Timezone:          s.timezoneOf(ctx, control, req.TenantInfo),
		Limit:             detailWorstRatedLimit,
	}
	days, err := s.feedback.DailySatisfaction(ctx, windowReq)
	if err != nil {
		return err
	}

	positive, negative := 0, 0
	for _, day := range days {
		positive += day.Positive
		negative += day.Negative
		detail.SatisfactionPoints = append(
			detail.SatisfactionPoints,
			&aifeedbackports.AgentFeedbackDay{
				Day:          day.Day,
				Positive:     day.Positive,
				Negative:     day.Negative,
				Satisfaction: satisfaction(day.Positive, day.Negative),
			},
		)
	}
	detail.Ratings = positive + negative
	detail.Satisfaction = satisfaction(positive, negative)

	worst, _, err := s.worstRated(ctx, windowReq, req.ViewerID)
	if err != nil {
		return err
	}
	detail.WorstRated = worst

	return nil
}

func (s *Service) worstRated(
	ctx context.Context,
	req aifeedbackports.AIFeedbackWindowRequest,
	viewerID pulid.ID,
) ([]*agentqualityports.AgentWorstRatedAnswer, bool, error) {
	limit := req.Limit
	req.Limit = limit + 1
	scores, err := s.feedback.WorstRated(ctx, req)
	if err != nil {
		return nil, false, err
	}
	hasNext := len(scores) > limit
	if hasNext {
		scores = scores[:limit]
	}

	rows := worstRatedRows(scores)
	if err = s.attachSamples(ctx, sampleScope{
		tenant: req.TenantInfo,
		viewer: viewerID,
	}, scores, rows); err != nil {
		return nil, false, err
	}

	answers := make([]*agentqualityports.AgentWorstRatedAnswer, 0, len(rows))
	for idx := range rows {
		answers = append(answers, rows[idx].answer)
	}

	return answers, hasNext, nil
}

func (s *Service) agentNames(
	ctx context.Context,
	tenant pagination.TenantInfo,
	samples map[pulid.ID]*aifeedback.Feedback,
) (map[pulid.ID]string, error) {
	seen := make(map[pulid.ID]struct{}, len(samples))
	ids := make([]pulid.ID, 0, len(samples))
	for _, sample := range samples {
		if sample.AgentDefinitionID == nil {
			continue
		}
		if _, dup := seen[*sample.AgentDefinitionID]; dup {
			continue
		}
		seen[*sample.AgentDefinitionID] = struct{}{}
		ids = append(ids, *sample.AgentDefinitionID)
	}
	names := make(map[pulid.ID]string, len(ids))
	if len(ids) == 0 {
		return names, nil
	}

	definitions, err := s.definitions.ListByIDs(
		ctx,
		agentdefinitionports.ListAgentDefinitionsByIDsRequest{
			IDs:        ids,
			TenantInfo: tenant,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, definition := range definitions {
		names[definition.ID] = definition.Name
	}

	return names, nil
}

type agentPageRequest struct {
	tenant         pagination.TenantInfo
	ids            []pulid.ID
	since          int64
	previousSince  int64
	includeRatings bool
	skipHistory    bool
	skipLatest     bool
}

type ratingTotals struct {
	Positive int
	Negative int
}

type agentPageData struct {
	history  map[pulid.ID][]*agentqualitysuite.SuiteRun
	latest   map[pulid.ID]*agentqualitysuite.SuiteRun
	current  map[pulid.ID]ratingTotals
	previous map[pulid.ID]ratingTotals
}

func (s *Service) agentPageData(ctx context.Context, req agentPageRequest) (*agentPageData, error) {
	data := &agentPageData{
		history:  make(map[pulid.ID][]*agentqualitysuite.SuiteRun, len(req.ids)),
		latest:   make(map[pulid.ID]*agentqualitysuite.SuiteRun, len(req.ids)),
		current:  make(map[pulid.ID]ratingTotals, len(req.ids)),
		previous: make(map[pulid.ID]ratingTotals, len(req.ids)),
	}

	if !req.skipHistory {
		history, err := s.suiteRuns.History(ctx, agentqualityports.AgentSuiteRunHistoryRequest{
			TenantInfo:         req.tenant,
			AgentDefinitionIDs: req.ids,
			Since:              req.since,
			PerAgent:           agentqualityports.AgentQualityHistoryPoints,
		})
		if err != nil {
			return nil, err
		}
		for _, run := range history {
			data.history[run.AgentDefinitionID] = append(data.history[run.AgentDefinitionID], run)
		}
	}

	if !req.skipLatest {
		latest, err := s.suiteRuns.Latest(ctx, agentqualityports.LatestAgentSuiteRunsRequest{
			TenantInfo:         req.tenant,
			AgentDefinitionIDs: req.ids,
		})
		if err != nil {
			return nil, err
		}
		for _, run := range latest {
			data.latest[run.AgentDefinitionID] = run
		}
	}

	if !req.includeRatings {
		return data, nil
	}

	if err := s.fillTotals(ctx, req, req.since, 0, data.current); err != nil {
		return nil, err
	}
	if err := s.fillTotals(ctx, req, req.previousSince, req.since, data.previous); err != nil {
		return nil, err
	}

	return data, nil
}

func (s *Service) fillTotals(
	ctx context.Context,
	req agentPageRequest,
	since, until int64,
	into map[pulid.ID]ratingTotals,
) error {
	totals, err := s.feedback.TotalsByAgent(ctx, aifeedbackports.AIFeedbackAgentTotalsRequest{
		TenantInfo:         req.tenant,
		AgentDefinitionIDs: req.ids,
		Since:              since,
		Until:              until,
	})
	if err != nil {
		return err
	}
	for _, total := range totals {
		into[total.AgentDefinitionID] = ratingTotals{
			Positive: total.Positive,
			Negative: total.Negative,
		}
	}

	return nil
}

func (s *Service) GetSuiteRun(
	ctx context.Context,
	id pulid.ID,
	tenant pagination.TenantInfo,
) (*agentqualitysuite.SuiteRun, error) {
	return s.suiteRuns.GetByID(ctx, agentqualityports.GetAgentSuiteRunRequest{
		ID:         id,
		TenantInfo: tenant,
	})
}
