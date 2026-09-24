package agentqualityservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/aifeedback"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/memtable"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	secondsPerDay         = int64(24 * 60 * 60)
	agentsCursorScope     = "agent-quality-agents"
	worstRatedCursorScope = "agent-worst-rated"
	suiteRunsCursorScope  = "agent-suite-runs"
	suiteCasesCursorScope = "agent-suite-run-cases"
	detailWorstRatedLimit = 5
)

func windowDays(requested int) (int, error) {
	if requested == 0 {
		return services.DefaultAgentQualityWindowDays, nil
	}
	if requested < 1 || requested > services.MaxAgentQualityWindowDays {
		return 0, errortypes.NewValidationError(
			"window",
			errortypes.ErrInvalid,
			"The window must be between 1 and {0} days",
			services.MaxAgentQualityWindowDays,
		)
	}

	return requested, nil
}

func pageSize(requested int) int {
	if requested <= 0 {
		return services.DefaultAgentQualityPageSize
	}

	return min(requested, services.MaxAgentQualityPageSize)
}

func decodeOffset(scope, after string) (int, error) {
	offset, err := pagination.DecodeOffsetCursor(scope, after)
	if err != nil {
		return 0, errortypes.NewValidationError(
			"after",
			errortypes.ErrInvalidFormat,
			"Cursor is invalid",
		)
	}

	return offset, nil
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
	req *services.AgentQualityOverviewRequest,
) (*services.AgentQualityOverview, error) {
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

	overview := &services.AgentQualityOverview{
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
			repositories.AIFeedbackAgentTotalsRequest{
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

	runs, err := s.suiteRuns.Overview(ctx, repositories.AgentSuiteRunOverviewRequest{
		TenantInfo: req.TenantInfo,
		Since:      since,
	})
	if err != nil {
		return nil, err
	}
	overview.SuiteRuns = runs.Runs
	overview.Regressions = runs.Regressions

	latest, err := s.suiteRuns.Latest(ctx, repositories.LatestAgentSuiteRunsRequest{
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
	overview.AgentsScored = len(scores)
	overview.QualityScore = meanPointer(scores)

	withCases, err := s.cases.ListAgentsWithActiveCases(
		ctx,
		repositories.ListAgentsWithActiveEvalCasesRequest{TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}
	overview.AgentsWithCases = len(withCases)

	spend, err := s.usage.EvaluationCost(ctx, repositories.AIUsageEvaluationCostRequest{
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

func qualityPoints(runs []*agentquality.SuiteRun) []*services.AgentQualityPoint {
	points := make([]*services.AgentQualityPoint, 0, len(runs))
	for _, run := range runs {
		if run.QualityScore == nil {
			continue
		}
		at := run.StartedAt
		if run.FinishedAt != nil {
			at = *run.FinishedAt
		}
		points = append(points, &services.AgentQualityPoint{
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
	req *services.AgentQualityRequest,
) (*services.AgentQualityDetail, error) {
	window, err := windowDays(req.WindowDays)
	if err != nil {
		return nil, err
	}

	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         req.AgentDefinitionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	now := s.now()
	since := now - int64(window)*secondsPerDay
	detail := &services.AgentQualityDetail{
		AgentDefinitionID:  definition.ID,
		AgentName:          definition.Name,
		Enabled:            definition.Enabled,
		WindowDays:         window,
		Since:              since,
		RatingsVisible:     req.IncludeRatings,
		SatisfactionPoints: []*services.AgentFeedbackDay{},
		WorstRated:         []*services.AgentWorstRatedAnswer{},
	}

	if req.IncludeRatings {
		if err = s.fillRatings(ctx, detail, req, since); err != nil {
			return nil, err
		}
	}

	history, err := s.suiteRuns.History(ctx, repositories.AgentSuiteRunHistoryRequest{
		TenantInfo:         req.TenantInfo,
		AgentDefinitionIDs: []pulid.ID{definition.ID},
		Since:              since,
		PerAgent:           services.AgentQualityHistoryPoints,
	})
	if err != nil {
		return nil, err
	}
	detail.QualityPoints = qualityPoints(history)

	last, err := s.suiteRuns.Last(ctx, repositories.LastAgentSuiteRunRequest{
		AgentDefinitionID: definition.ID,
		TenantInfo:        req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	detail.LastSuiteRun = last

	cases, err := s.cases.ListSamplingCases(ctx, repositories.ListEvalCaseSamplingRequest{
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
	detail *services.AgentQualityDetail,
	req *services.AgentQualityRequest,
	since int64,
) error {
	control, err := s.control(ctx, req.TenantInfo)
	if err != nil {
		return err
	}

	windowReq := repositories.AIFeedbackWindowRequest{
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
		detail.SatisfactionPoints = append(detail.SatisfactionPoints, &services.AgentFeedbackDay{
			Day:          day.Day,
			Positive:     day.Positive,
			Negative:     day.Negative,
			Satisfaction: satisfaction(day.Positive, day.Negative),
		})
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
	req repositories.AIFeedbackWindowRequest,
	viewerID pulid.ID,
) ([]*services.AgentWorstRatedAnswer, bool, error) {
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

	answers := make([]*services.AgentWorstRatedAnswer, 0, len(rows))
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

	definitions, err := s.definitions.ListByIDs(ctx, repositories.ListAgentDefinitionsByIDsRequest{
		IDs:        ids,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}
	for _, definition := range definitions {
		names[definition.ID] = definition.Name
	}

	return names, nil
}

func (s *Service) ListWorstRated(
	ctx context.Context,
	req *services.ListAgentWorstRatedRequest,
) (*services.AgentWorstRatedPage, error) {
	return s.ListWorstRatedTable(ctx, &services.ListAgentWorstRatedTableRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: req.AgentDefinitionID,
		WindowDays:        req.WindowDays,
		ViewerID:          req.ViewerID,
		Table: memtable.Request{
			First: req.First,
			After: req.After,
		},
	})
}

func (s *Service) ListAgents(
	ctx context.Context,
	req *services.ListAgentQualityAgentsRequest,
) (*services.AgentQualityAgentPage, error) {
	return s.ListAgentTable(ctx, &services.ListAgentQualityAgentTableRequest{
		TenantInfo:     req.TenantInfo,
		WindowDays:     req.WindowDays,
		IncludeRatings: req.IncludeRatings,
		Table: memtable.Request{
			First:             req.First,
			After:             req.After,
			IncludeTotalCount: req.IncludeTotalCount,
		},
	})
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
	history  map[pulid.ID][]*agentquality.SuiteRun
	latest   map[pulid.ID]*agentquality.SuiteRun
	current  map[pulid.ID]ratingTotals
	previous map[pulid.ID]ratingTotals
}

func (s *Service) agentPageData(ctx context.Context, req agentPageRequest) (*agentPageData, error) {
	data := &agentPageData{
		history:  make(map[pulid.ID][]*agentquality.SuiteRun, len(req.ids)),
		latest:   make(map[pulid.ID]*agentquality.SuiteRun, len(req.ids)),
		current:  make(map[pulid.ID]ratingTotals, len(req.ids)),
		previous: make(map[pulid.ID]ratingTotals, len(req.ids)),
	}

	if !req.skipHistory {
		history, err := s.suiteRuns.History(ctx, repositories.AgentSuiteRunHistoryRequest{
			TenantInfo:         req.tenant,
			AgentDefinitionIDs: req.ids,
			Since:              req.since,
			PerAgent:           services.AgentQualityHistoryPoints,
		})
		if err != nil {
			return nil, err
		}
		for _, run := range history {
			data.history[run.AgentDefinitionID] = append(data.history[run.AgentDefinitionID], run)
		}
	}

	if !req.skipLatest {
		latest, err := s.suiteRuns.Latest(ctx, repositories.LatestAgentSuiteRunsRequest{
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
	totals, err := s.feedback.TotalsByAgent(ctx, repositories.AIFeedbackAgentTotalsRequest{
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

func (s *Service) ListSuiteRuns(
	ctx context.Context,
	req *services.ListAgentSuiteRunsInput,
) (*services.AgentSuiteRunConnection, error) {
	offset, err := decodeOffset(suiteRunsCursorScope, req.After)
	if err != nil {
		return nil, err
	}
	for _, status := range req.Statuses {
		if !status.IsValid() {
			return nil, errortypes.NewValidationError(
				"statuses",
				errortypes.ErrInvalid,
				"Status is invalid",
			)
		}
	}

	page, err := s.suiteRuns.List(ctx, repositories.ListAgentSuiteRunsRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: req.AgentDefinitionID,
		Statuses:          req.Statuses,
		Limit:             pageSize(req.First),
		Offset:            offset,
		IncludeTotalCount: req.IncludeTotalCount,
	})
	if err != nil {
		return nil, err
	}

	connection := &services.AgentSuiteRunConnection{
		Edges:       make([]*services.AgentSuiteRunEdge, 0, len(page.Items)),
		HasNextPage: page.HasNextPage,
		TotalCount:  page.TotalCount,
	}
	for idx, run := range page.Items {
		connection.Edges = append(connection.Edges, &services.AgentSuiteRunEdge{
			Node:   run,
			Cursor: pagination.EncodeOffsetCursor(suiteRunsCursorScope, offset+idx+1),
		})
	}

	return connection, nil
}

func (s *Service) ListSuiteRunCases(
	ctx context.Context,
	req *services.ListAgentSuiteRunCasesRequest,
) (*services.AgentSuiteRunCasePage, error) {
	after, err := decodeOffset(suiteCasesCursorScope, req.After)
	if err != nil {
		return nil, err
	}

	run, err := s.suiteRuns.GetByID(ctx, repositories.GetAgentSuiteRunRequest{
		ID:         req.SuiteRunID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	limit := pageSize(req.First)
	evaluations, err := s.evaluations.ListBySuiteRun(ctx, repositories.ListSuiteEvaluationsRequest{
		TenantInfo:   req.TenantInfo,
		SuiteRunID:   run.ID,
		AfterOrdinal: after,
		Limit:        limit + 1,
	})
	if err != nil {
		return nil, err
	}

	page := &services.AgentSuiteRunCasePage{HasNextPage: len(evaluations) > limit}
	if page.HasNextPage {
		evaluations = evaluations[:limit]
	}
	if req.IncludeTotalCount {
		total := run.CasesTotal
		page.TotalCount = &total
	}
	page.Edges = make([]*services.AgentSuiteRunCaseEdge, 0, len(evaluations))
	for _, evaluation := range evaluations {
		ordinal := 0
		if evaluation.SuiteOrdinal != nil {
			ordinal = *evaluation.SuiteOrdinal
		}
		page.Edges = append(page.Edges, &services.AgentSuiteRunCaseEdge{
			Node:   evaluation,
			Cursor: pagination.EncodeOffsetCursor(suiteCasesCursorScope, ordinal),
		})
	}

	return page, nil
}

func (s *Service) GetSuiteRun(
	ctx context.Context,
	id pulid.ID,
	tenant pagination.TenantInfo,
) (*agentquality.SuiteRun, error) {
	return s.suiteRuns.GetByID(ctx, repositories.GetAgentSuiteRunRequest{
		ID:         id,
		TenantInfo: tenant,
	})
}
