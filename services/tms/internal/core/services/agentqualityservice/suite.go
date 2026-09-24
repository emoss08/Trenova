package agentqualityservice

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentscoring"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const (
	EventQualityRegression = "agent.quality_regression"
	regressionNoticeSource = "agent-quality"
	maxRegressionNotices   = 50
	suiteReadPage          = 200
	caseSourcePage         = 2000
	historyLookbackSeconds = int64(180 * 24 * 60 * 60)
	budgetStoppedCaseNote  = "Not replayed: the evaluation budget ran out."
	failedSuiteCaseNote    = "Not replayed: the suite run failed."
)

func scoredStatuses() []agentquality.SuiteRunStatus {
	return []agentquality.SuiteRunStatus{
		agentquality.SuiteRunStatusCompleted,
		agentquality.SuiteRunStatusBudgetStopped,
	}
}

type SuiteCase struct {
	EvaluationID pulid.ID               `json:"evaluationId"`
	Ordinal      int                    `json:"ordinal"`
	Status       agent.EvaluationStatus `json:"status"`
}

type ListSuiteCasesRequest struct {
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	SuiteRunID   pulid.ID              `json:"suiteRunId"`
	AfterOrdinal int                   `json:"afterOrdinal"`
	Limit        int                   `json:"limit"`
}

func (s *Service) ListSuiteCases(
	ctx context.Context,
	req *ListSuiteCasesRequest,
) ([]SuiteCase, error) {
	evaluations, err := s.evaluations.ListBySuiteRun(ctx, repositories.ListSuiteEvaluationsRequest{
		TenantInfo:   req.TenantInfo,
		SuiteRunID:   req.SuiteRunID,
		AfterOrdinal: req.AfterOrdinal,
		Limit:        req.Limit,
	})
	if err != nil {
		return nil, err
	}

	cases := make([]SuiteCase, 0, len(evaluations))
	for _, evaluation := range evaluations {
		ordinal := 0
		if evaluation.SuiteOrdinal != nil {
			ordinal = *evaluation.SuiteOrdinal
		}
		cases = append(cases, SuiteCase{
			EvaluationID: evaluation.ID,
			Ordinal:      ordinal,
			Status:       evaluation.Status,
		})
	}

	return cases, nil
}

type CheckBudgetRequest struct {
	TenantInfo pagination.TenantInfo      `json:"tenantInfo"`
	DayStart   int64                      `json:"dayStart"`
	MonthStart int64                      `json:"monthStart"`
	Settings   agentquality.SuiteSettings `json:"settings"`
}

func (s *Service) CheckBudget(
	ctx context.Context,
	req *CheckBudgetRequest,
) (*agentquality.BudgetDecision, error) {
	nightly, err := s.usage.EvaluationCost(ctx, repositories.AIUsageEvaluationCostRequest{
		TenantInfo: req.TenantInfo,
		Since:      req.DayStart,
	})
	if err != nil {
		return nil, err
	}
	monthly, err := s.usage.EvaluationCost(ctx, repositories.AIUsageEvaluationCostRequest{
		TenantInfo: req.TenantInfo,
		Since:      req.MonthStart,
	})
	if err != nil {
		return nil, err
	}

	decision := agentquality.CheckBudget(agentquality.BudgetInput{
		NightlySpent: nightly.CostUSD,
		MonthlySpent: monthly.CostUSD,
		NightlyCap:   req.Settings.NightlyBudgetUSD,
		MonthlyCap:   req.Settings.MonthlyBudgetUSD,
	})

	return &decision, nil
}

type suiteOutcome struct {
	evaluations   []*agent.Evaluation
	scored        []agentscoring.CaseResult
	deterministic []agentscoring.CaseResult
	judged        []float64
	passed        int
	failed        int
	skipped       int
	hardFailures  int
	judgeable     []pulid.ID
}

func (s *Service) suiteEvaluationsOf(
	ctx context.Context,
	tenant pagination.TenantInfo,
	suiteRunID pulid.ID,
) ([]*agent.Evaluation, error) {
	all := make([]*agent.Evaluation, 0, suiteReadPage)
	after := 0
	for {
		page, err := s.evaluations.ListBySuiteRun(ctx, repositories.ListSuiteEvaluationsRequest{
			TenantInfo:   tenant,
			SuiteRunID:   suiteRunID,
			AfterOrdinal: after,
			Limit:        suiteReadPage,
		})
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) < suiteReadPage {
			return all, nil
		}
		last := page[len(page)-1].SuiteOrdinal
		if last == nil {
			return all, nil
		}
		after = *last
	}
}

func (s *Service) caseSources(
	ctx context.Context,
	tenant pagination.TenantInfo,
	agentID pulid.ID,
) (map[pulid.ID]agentquality.CaseSource, error) {
	cases, err := s.cases.ListByAgent(ctx, repositories.ListAgentEvalCasesRequest{
		AgentDefinitionID: agentID,
		TenantInfo:        tenant,
		Limit:             caseSourcePage,
	})
	if err != nil {
		return nil, err
	}

	sources := make(map[pulid.ID]agentquality.CaseSource, len(cases))
	for _, evalCase := range cases {
		sources[evalCase.ID] = evalCase.Source
	}

	return sources, nil
}

func outcomeOf(
	evaluations []*agent.Evaluation,
	sources map[pulid.ID]agentquality.CaseSource,
) *suiteOutcome {
	out := &suiteOutcome{evaluations: evaluations}
	for _, evaluation := range evaluations {
		switch {
		case evaluation.Status == agent.EvaluationStatusCompleted && evaluation.Checks != nil:
			checks := evaluation.Checks
			caseID := pulid.Nil
			if evaluation.EvalCaseID != nil {
				caseID = *evaluation.EvalCaseID
			}
			source := sources[caseID]
			if !source.IsValid() {
				source = agentquality.CaseSourceCurated
			}
			deterministic := checks.Deterministic
			if checks.HardFailure {
				deterministic = 0
				out.hardFailures++
			}
			final := checks.Final
			if evaluation.CaseScore != nil {
				final = *evaluation.CaseScore
			}
			out.scored = append(out.scored, agentscoring.CaseResult{
				CaseID:      caseID,
				Source:      source,
				Score:       final,
				Passed:      checks.Passed,
				HardFailure: checks.HardFailure,
			})
			out.deterministic = append(out.deterministic, agentscoring.CaseResult{
				CaseID:      caseID,
				Source:      source,
				Score:       deterministic,
				Passed:      checks.Passed,
				HardFailure: checks.HardFailure,
			})
			if evaluation.Judge != nil {
				out.judged = append(out.judged, evaluation.Judge.Score)
			} else if !checks.HardFailure {
				out.judgeable = append(out.judgeable, evaluation.ID)
			}
			if checks.Passed {
				out.passed++
			} else {
				out.failed++
			}
		case evaluation.Status == agent.EvaluationStatusFailed,
			evaluation.Status == agent.EvaluationStatusCompleted:
			out.failed++
		default:
			out.skipped++
		}
	}

	return out
}

func scorePointer(results []agentscoring.CaseResult) *float64 {
	if len(results) == 0 {
		return nil
	}
	score := agentscoring.ScoreSuite(results).Score

	return &score
}

func meanPointer(values []float64) *float64 {
	if len(values) == 0 {
		return nil
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	mean := total / float64(len(values))

	return &mean
}

type ScoreSuiteRequest struct {
	TenantInfo pagination.TenantInfo      `json:"tenantInfo"`
	SuiteRunID pulid.ID                   `json:"suiteRunId"`
	SampleSeed int64                      `json:"sampleSeed"`
	Settings   agentquality.SuiteSettings `json:"settings"`
}

type ScoredSuite struct {
	Scored             int        `json:"scored"`
	DeterministicScore *float64   `json:"deterministicScore,omitempty"`
	JudgeCases         []pulid.ID `json:"judgeCases"`
}

func (s *Service) ScoreSuite(ctx context.Context, req *ScoreSuiteRequest) (*ScoredSuite, error) {
	run, err := s.suiteRuns.GetByID(ctx, repositories.GetAgentSuiteRunRequest{
		ID:         req.SuiteRunID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	evaluations, err := s.suiteEvaluationsOf(ctx, req.TenantInfo, run.ID)
	if err != nil {
		return nil, err
	}
	sources, err := s.caseSources(ctx, req.TenantInfo, run.AgentDefinitionID)
	if err != nil {
		return nil, err
	}

	outcome := outcomeOf(evaluations, sources)
	scored := &ScoredSuite{
		Scored:             len(outcome.scored),
		DeterministicScore: scorePointer(outcome.deterministic),
		JudgeCases:         []pulid.ID{},
	}
	if req.Settings.JudgeEnabled && s.completion != nil {
		scored.JudgeCases = agentquality.JudgeSample(
			outcome.judgeable,
			req.SampleSeed,
			req.Settings.JudgeSampleRate,
		)
	}

	if !run.Status.Terminal() {
		run.DeterministicScore = scored.DeterministicScore
		run.CasesPassed = outcome.passed
		run.CasesFailed = outcome.failed
		run.CasesSkipped = outcome.skipped
		run.HardFailures = outcome.hardFailures
		if _, err = s.suiteRuns.Update(ctx, run); err != nil {
			return nil, err
		}
	}

	return scored, nil
}

type JudgeCaseRequest struct {
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	EvaluationID pulid.ID              `json:"evaluationId"`
}

type JudgedCase struct {
	Judged bool     `json:"judged"`
	Score  *float64 `json:"score,omitempty"`
	Note   string   `json:"note,omitempty"`
}

func (s *Service) JudgeCase(ctx context.Context, req *JudgeCaseRequest) (*JudgedCase, error) {
	if s.completion == nil {
		return &JudgedCase{Note: "No completion service is available to judge with."}, nil
	}

	evaluation, err := s.evaluations.GetByID(ctx, repositories.GetAgentEvaluationByIDRequest{
		ID:         req.EvaluationID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if evaluation.Judge != nil {
		score := evaluation.Judge.Score

		return &JudgedCase{Judged: true, Score: &score}, nil
	}
	if evaluation.Checks == nil || evaluation.EvalCaseID == nil ||
		evaluation.Status != agent.EvaluationStatusCompleted {
		return &JudgedCase{Note: "The case has no scored answer to judge."}, nil
	}
	if evaluation.Checks.HardFailure {
		return &JudgedCase{Note: "A hard check failed; the judge does not override it."}, nil
	}

	evalCase, err := s.cases.GetByID(ctx, repositories.GetAgentEvalCaseByIDRequest{
		ID:         *evaluation.EvalCaseID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	result, err := s.completion.CompleteStructured(ctx, &services.StructuredCompletionRequest{
		TenantInfo:   req.TenantInfo,
		Task:         aiprovider.TaskEvaluationJudge,
		System:       judgeSystemPrompt,
		Context:      judgeContext(evalCase, evaluation.Reply),
		OutputSchema: judgeSchema(),
		SchemaName:   judgeSchemaName,
		MaxTokens:    judgeMaxTokens,
		Attribution: services.AIUsageAttribution{
			AgentDefinitionID: evaluation.AgentDefinitionID,
			RunID:             evaluation.ID,
			Purpose:           services.AIUsagePurposeEvaluation,
			Feature:           aiusage.FeatureAgentEvaluation,
		},
	})
	if errors.Is(err, services.ErrNoProviderConfigured) {
		return &JudgedCase{Note: "No provider is assigned to the evaluation judge."}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ask the judge: %w", err)
	}

	decoded, err := decodeJudgement(result.Text)
	if err != nil {
		s.l.Warn("rejected a judgement that could not be read",
			zap.String("evaluation", evaluation.ID.String()),
			zap.String("model", result.ModelIdentifier),
			zap.Error(err),
		)

		return &JudgedCase{Note: "The judge's answer could not be read."}, nil
	}

	verdict := &agent.JudgeVerdict{
		Score:     decoded.Score,
		Rationale: decoded.Rationale,
		Model:     result.ModelIdentifier,
		JudgedAt:  s.now(),
	}
	final := applyJudgement(evaluation.Checks, verdict)
	evaluation.Judge = verdict
	evaluation.CaseScore = &final
	if _, err = s.evaluations.Update(ctx, evaluation); err != nil {
		return nil, fmt.Errorf("keep the judgement: %w", err)
	}

	score := decoded.Score

	return &JudgedCase{Judged: true, Score: &score}, nil
}

type FinalizeSuiteRequest struct {
	TenantInfo pagination.TenantInfo      `json:"tenantInfo"`
	SuiteRunID pulid.ID                   `json:"suiteRunId"`
	Settings   agentquality.SuiteSettings `json:"settings"`
	StopReason string                     `json:"stopReason,omitempty"`
}

type FinalizedSuite struct {
	Status       agentquality.SuiteRunStatus `json:"status"`
	QualityScore *float64                    `json:"qualityScore,omitempty"`
	Regression   bool                        `json:"regression"`
	HardFailures int                         `json:"hardFailures"`
}

func finalizedFrom(run *agentquality.SuiteRun) *FinalizedSuite {
	return &FinalizedSuite{
		Status:       run.Status,
		QualityScore: run.QualityScore,
		Regression:   run.Regression,
		HardFailures: run.HardFailures,
	}
}

func (s *Service) FinalizeSuite(
	ctx context.Context,
	req *FinalizeSuiteRequest,
) (*FinalizedSuite, error) {
	run, err := s.suiteRuns.GetByID(ctx, repositories.GetAgentSuiteRunRequest{
		ID:         req.SuiteRunID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if run.Status.Terminal() {
		return finalizedFrom(run), nil
	}

	if req.StopReason != "" {
		if _, err = s.evaluations.SkipPendingBySuiteRun(
			ctx,
			repositories.SkipPendingSuiteEvaluationsRequest{
				TenantInfo: req.TenantInfo,
				SuiteRunID: run.ID,
				Reason:     budgetStoppedCaseNote,
				At:         s.now(),
			},
		); err != nil {
			return nil, err
		}
	}

	evaluations, err := s.suiteEvaluationsOf(ctx, req.TenantInfo, run.ID)
	if err != nil {
		return nil, err
	}
	sources, err := s.caseSources(ctx, req.TenantInfo, run.AgentDefinitionID)
	if err != nil {
		return nil, err
	}
	outcome := outcomeOf(evaluations, sources)

	run.CasesPassed = outcome.passed
	run.CasesFailed = outcome.failed
	run.CasesSkipped = outcome.skipped
	run.HardFailures = outcome.hardFailures
	run.DeterministicScore = scorePointer(outcome.deterministic)
	run.JudgeScore = meanPointer(outcome.judged)
	run.QualityScore = scorePointer(outcome.scored)
	if cost, costErr := s.usage.EvaluationCost(ctx, repositories.AIUsageEvaluationCostRequest{
		TenantInfo: req.TenantInfo,
		SuiteRunID: run.ID,
	}); costErr != nil {
		s.l.Warn("could not read what a suite run cost", zap.Error(costErr))
	} else {
		run.CostUSD = cost.CostUSD
	}

	status, comments := finalStatus(outcome, req.StopReason)
	if status.Scored() {
		regression, regressionErr := s.detectRegression(ctx, run, outcome, req.Settings)
		if regressionErr != nil {
			return nil, regressionErr
		}
		run.Regression = regression.Regressed
		if regression.Compared > 0 {
			median := regression.Median
			run.BaselineScore = &median
		}
		comments = joinComments(comments, regressionComment(regression))
	}

	run.Finish(status, stringutils.Ellipsize(comments, agentquality.MaxSuiteComments), s.now())
	s.announce(ctx, run)

	updated, err := s.suiteRuns.Update(ctx, run)
	if err != nil {
		return nil, err
	}

	return finalizedFrom(updated), nil
}

func finalStatus(outcome *suiteOutcome, stopReason string) (agentquality.SuiteRunStatus, string) {
	switch {
	case stopReason != "":
		return agentquality.SuiteRunStatusBudgetStopped, stopReason
	case len(outcome.scored) == 0:
		return agentquality.SuiteRunStatusFailed,
			"No case could be replayed, so the agent was not scored."
	default:
		return agentquality.SuiteRunStatusCompleted, ""
	}
}

func joinComments(parts ...string) string {
	joined := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		if joined != "" {
			joined += " "
		}
		joined += part
	}

	return joined
}

func regressionComment(regression agentscoring.Regression) string {
	if !regression.Regressed {
		return ""
	}

	comment := ""
	if regression.ScoreDropped {
		comment = "The score fell " + strconv.Itoa(int(regression.Drop*100+0.5)) +
			" points below the recent median."
	}
	if count := len(regression.NewHardFailures); count > 0 {
		noun := "cases that passed before now break"
		if count == 1 {
			noun = "case that passed before now breaks"
		}
		comment = joinComments(comment, strconv.Itoa(count)+" "+noun+" a hard check.")
	}

	return comment
}

func (s *Service) detectRegression(
	ctx context.Context,
	run *agentquality.SuiteRun,
	outcome *suiteOutcome,
	settings agentquality.SuiteSettings,
) (agentscoring.Regression, error) {
	previous, err := s.suiteRuns.History(ctx, repositories.AgentSuiteRunHistoryRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: run.OrganizationID,
			BuID:  run.BusinessUnitID,
		},
		AgentDefinitionIDs: []pulid.ID{run.AgentDefinitionID},
		Since:              run.StartedAt - historyLookbackSeconds,
		PerAgent:           agentscoring.DefaultRegressionWindow + 1,
	})
	if err != nil {
		return agentscoring.Regression{}, err
	}

	history := make([]agentscoring.SuiteRun, 0, len(previous))
	for _, prior := range previous {
		if prior.ID == run.ID || prior.StartedAt > run.StartedAt || prior.QualityScore == nil {
			continue
		}
		completedAt := prior.StartedAt
		if prior.FinishedAt != nil {
			completedAt = *prior.FinishedAt
		}
		history = append(history, agentscoring.SuiteRun{
			Score:       *prior.QualityScore,
			Cases:       prior.CasesPassed + prior.CasesFailed,
			CompletedAt: completedAt,
		})
	}

	baseline, err := s.baselineResults(ctx, run)
	if err != nil {
		return agentscoring.Regression{}, err
	}

	minCases := settings.MinCases
	if minCases <= 0 {
		minCases = agentquality.DefaultMinCases
	}
	threshold := settings.RegressionThreshold
	if threshold <= 0 {
		threshold = agentquality.DefaultRegressionThreshold
	}

	return agentscoring.DetectRegression(agentscoring.RegressionInput{
		Current:  agentscoring.ScoreSuite(outcome.scored),
		Results:  outcome.scored,
		History:  history,
		Baseline: baseline,
	}, agentscoring.RegressionPolicy{
		Threshold: threshold,
		MinCases:  minCases,
		Window:    agentscoring.DefaultRegressionWindow,
	}), nil
}

func (s *Service) baselineResults(
	ctx context.Context,
	run *agentquality.SuiteRun,
) ([]agentscoring.CaseResult, error) {
	if run.BaselineRunID == nil || run.BaselineRunID.IsNil() {
		return nil, nil
	}

	tenant := pagination.TenantInfo{OrgID: run.OrganizationID, BuID: run.BusinessUnitID}
	evaluations, err := s.suiteEvaluationsOf(ctx, tenant, *run.BaselineRunID)
	if err != nil {
		return nil, err
	}

	return outcomeOf(evaluations, nil).scored, nil
}

func (s *Service) announce(ctx context.Context, run *agentquality.SuiteRun) {
	tenant := pagination.TenantInfo{OrgID: run.OrganizationID, BuID: run.BusinessUnitID}
	if s.watchtower != nil && run.BaselineRunID != nil && run.BaselineRunID.IsNotNil() {
		s.watchtower.Resolve(
			ctx,
			tenant,
			watchtower.SourceAgentQualityRegression,
			run.BaselineRunID.String(),
		)
	}
	if !run.Regression {
		return
	}

	name := s.agentName(ctx, tenant, run.AgentDefinitionID)
	item := watchtowersources.DescribeQualityRegression(run, name)
	if s.watchtower != nil {
		s.watchtower.Upsert(ctx, item)
	}
	s.notifyRegression(ctx, run, item)
}

func (s *Service) agentName(
	ctx context.Context,
	tenant pagination.TenantInfo,
	agentID pulid.ID,
) string {
	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         agentID,
		TenantInfo: tenant,
	})
	if err != nil {
		s.l.Warn("could not name the agent behind a regression", zap.Error(err))

		return ""
	}

	return definition.Name
}

func (s *Service) notifyRegression(
	ctx context.Context,
	run *agentquality.SuiteRun,
	item services.WatchtowerItemInput,
) {
	if s.notifier == nil {
		return
	}

	priority := notification.PriorityMedium
	if item.Severity == watchtower.SeverityCritical {
		priority = notification.PriorityHigh
	}
	correlation := run.ID.String()
	changes := make([]string, 0, len(run.FingerprintChanges))
	for _, change := range run.FingerprintChanges {
		changes = append(changes, string(change.Field))
	}

	if _, err := s.notifier.NotifyPermitted(ctx, notificationservice.NotifyPermittedRequest{
		Tenant:      item.TenantInfo,
		Resource:    permission.ResourceAgentControl,
		Operation:   permission.OpUpdate,
		Limit:       maxRegressionNotices,
		Now:         s.now(),
		DedupeSince: max(run.StartedAt-1, 1),
		Notification: notification.Notification{
			EventType:     EventQualityRegression,
			Priority:      priority,
			Title:         item.Title,
			Message:       item.Summary,
			Source:        regressionNoticeSource,
			CorrelationID: &correlation,
			Data: map[string]any{
				"link":              item.Path,
				"suiteRunId":        correlation,
				"agentDefinitionId": run.AgentDefinitionID.String(),
				"severity":          string(item.Severity),
				"changes":           changes,
			},
			RelatedEntities: map[string]any{"agentSuiteRunId": correlation},
		},
	}); err != nil {
		s.l.Warn("could not tell anyone an agent regressed",
			zap.String("suiteRun", correlation), zap.Error(err))
	}
}

type FailSuiteRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	SuiteRunID pulid.ID              `json:"suiteRunId"`
	Error      string                `json:"error"`
}

func (s *Service) FailSuite(ctx context.Context, req *FailSuiteRequest) error {
	run, err := s.suiteRuns.GetByID(ctx, repositories.GetAgentSuiteRunRequest{
		ID:         req.SuiteRunID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}
	if run.Status.Terminal() {
		return nil
	}

	return s.fail(ctx, run, req.Error)
}

func (s *Service) fail(ctx context.Context, run *agentquality.SuiteRun, reason string) error {
	now := s.now()
	tenant := pagination.TenantInfo{OrgID: run.OrganizationID, BuID: run.BusinessUnitID}
	if _, err := s.evaluations.SkipPendingBySuiteRun(
		ctx,
		repositories.SkipPendingSuiteEvaluationsRequest{
			TenantInfo: tenant,
			SuiteRunID: run.ID,
			Reason:     failedSuiteCaseNote,
			At:         now,
		},
	); err != nil {
		return err
	}

	run.Finish(
		agentquality.SuiteRunStatusFailed,
		stringutils.Ellipsize(reason, agentquality.MaxSuiteComments),
		now,
	)
	if _, err := s.suiteRuns.Update(ctx, run); err != nil {
		return fmt.Errorf("record why a suite run failed: %w", err)
	}

	return nil
}
