package aianalyticsrepository

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision/agentdecisiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrunstep/agentrunstepdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentsuggestion"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentsuggestion/agentsuggestiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics/aianalyticsdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aifeedback"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aifeedback/aifeedbackdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/settingversion"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/settingversion/settingversiondb"
)

const (
	colUnitID = "unit_id"
)

var durationFractions = []float64{0.25, 0.5, 0.75, 0.95}

func unitScope(w *aianalyticsdb.WorkUnitsTable, filter *aianalyticsports.Filter) []keel.Pred {
	return []keel.Pred{
		keel.When(
			filter.AgentDefinitionID.IsNotNil(),
			w.AgentDefinitionID.Eq(filter.AgentDefinitionID),
		),
		keel.When(filter.Task != "", w.Task.Eq(filter.Task)),
	}
}

func unitWindow(w *aianalyticsdb.WorkUnitsTable, filter *aianalyticsports.Filter) []keel.Pred {
	return append(unitScope(w, filter), w.FinishedAt.Gte(filter.Since))
}

func hourScope(h *aianalyticsdb.ProviderHoursTable, filter *aianalyticsports.Filter) []keel.Pred {
	return []keel.Pred{
		h.Hour.Gte(ceilHour(filter.Since)),
		keel.When(
			filter.AgentDefinitionID.IsNotNil(),
			h.AgentDefinitionID.Eq(filter.AgentDefinitionID.String()),
		),
		keel.When(filter.Task != "", h.Task.Eq(filter.Task)),
	}
}

func toolScope(t *aianalyticsdb.UnitToolsTable, filter *aianalyticsports.Filter) []keel.Pred {
	return []keel.Pred{
		t.FinishedAt.Gte(filter.Since),
		keel.When(
			filter.AgentDefinitionID.IsNotNil(),
			t.AgentDefinitionID.Eq(filter.AgentDefinitionID.String()),
		),
		keel.When(filter.Task != "", t.Task.Eq(filter.Task)),
	}
}

func providerKey(w *aianalyticsdb.WorkUnitsTable) keel.StringExpr {
	return w.ProviderID.Coalesce("")
}

func deploymentColumns(w *aianalyticsdb.WorkUnitsTable) []keel.Expr {
	return []keel.Expr{w.ModelKey, providerKey(w)}
}

func groupColumns(w *aianalyticsdb.WorkUnitsTable, grouping aianalytics.Grouping) []keel.Expr {
	switch grouping {
	case aianalytics.GroupingProvider:
		return []keel.Expr{keel.Text(""), providerKey(w)}
	case aianalytics.GroupingDeployment:
		return deploymentColumns(w)
	default:
		return []keel.Expr{w.ModelKey, keel.Text("")}
	}
}

func completedUnit(w *aianalyticsdb.WorkUnitsTable) keel.Pred {
	return w.Outcome.Eq(aianalytics.OutcomeCompleted)
}

func summed(e keel.Expr) keel.NumericExpr {
	return keel.NumericOf(keel.Sum(e)).Coalesce(0)
}

func outcomeCounts(w *aianalyticsdb.WorkUnitsTable) []keel.Expr {
	return []keel.Expr{
		keel.Count(),
		keel.Count().Filter(completedUnit(w)),
		keel.Count().Filter(w.Outcome.Eq(aianalytics.OutcomeHandedOff)),
		keel.Count().Filter(w.Outcome.Eq(aianalytics.OutcomeStopped)),
		keel.Count().Filter(w.Outcome.Eq(aianalytics.OutcomeFailed)),
		summed(w.ModelCalls),
		summed(w.FailedCalls),
		summed(w.ToolCalls),
		summed(w.ToolFailuresModel),
		summed(w.ToolFailuresTool),
		keel.CastText(summed(w.CostUSD)),
		w.CostUSD.Count(),
		summed(w.InputTokens),
		summed(w.OutputTokens),
		summed(w.CacheReadTokens),
		summed(w.CacheWriteTokens),
		summed(w.AdvisorCalls),
		summed(w.AdvisorPromptTokens),
		summed(w.AdvisorOutputTokens),
		keel.CastText(summed(w.AdvisorCostUSD)),
		keel.Int64Of(w.FinishedAt.Max()).Coalesce(0),
	}
}

func durationPercentiles(w *aianalyticsdb.WorkUnitsTable, filter keel.Pred) []keel.Expr {
	out := make([]keel.Expr, 0, len(durationFractions))
	for _, fraction := range durationFractions {
		percentile := keel.PercentileCont(fraction, w.DurationMs.Asc())
		if filter != nil {
			percentile = percentile.Filter(filter)
		}
		out = append(out, percentile)
	}

	return out
}

func totalsQuery(filter *aianalyticsports.Filter) *keel.Query[*aianalytics.WorkUnit] {
	w := aianalyticsdb.WorkUnits
	picks := outcomeCounts(w)
	picks = append(picks, keel.Count().Filter(completedUnit(w)))
	picks = append(picks, durationPercentiles(w, completedUnit(w))...)

	return w.Pick(picks...).Where(unitWindow(w, filter)...)
}

func deploymentsQuery(filter *aianalyticsports.Filter) *keel.Query[*aianalytics.WorkUnit] {
	w := aianalyticsdb.WorkUnits
	keys := deploymentColumns(w)

	return w.Pick(append(keys, outcomeCounts(w)...)...).
		Where(unitWindow(w, filter)...).
		Group(keys...)
}

func outcomeDurationsQuery(
	filter *aianalyticsports.Filter,
	grouping aianalytics.Grouping,
) *keel.Query[*aianalytics.WorkUnit] {
	w := aianalyticsdb.WorkUnits
	keys := append(groupColumns(w, grouping), w.Outcome)
	picks := append(append([]keel.Expr{}, keys...), keel.Count())

	return w.Pick(append(picks, durationPercentiles(w, nil)...)...).
		Where(unitWindow(w, filter)...).
		Group(keys...)
}

func trendBucket(w *aianalyticsdb.WorkUnitsTable, starts []int64) keel.Expr {
	if len(starts) == 1 {
		return keel.Int64(starts[0])
	}
	bucket := keel.CaseInt64()
	for index := len(starts) - 1; index > 0; index-- {
		bucket = bucket.When(w.FinishedAt.Gte(starts[index]), keel.Int64(starts[index]))
	}

	return bucket.Else(keel.Int64(starts[0]))
}

func trendQuery(req *aianalyticsports.TrendRequest) *keel.Query[*aianalytics.WorkUnit] {
	w := aianalyticsdb.WorkUnits
	keys := append(groupColumns(w, req.Grouping), trendBucket(w, req.BucketStarts))

	return w.Pick(append(append([]keel.Expr{}, keys...),
		keel.Count(),
		keel.Count().Filter(completedUnit(w)),
		summed(w.ToolCalls),
		summed(w.ToolFailuresModel.Add(w.ToolFailuresTool)),
		keel.PercentileCont(0.5, w.DurationMs.Asc()).Filter(completedUnit(w)),
	)...).
		Where(append(unitWindow(w, &req.Filter), w.FinishedAt.Gte(req.BucketStarts[0]))...).
		Group(keys...)
}

func toolCellsQuery(filter *aianalyticsports.Filter) *keel.Query[*aianalytics.UnitTool] {
	t := aianalyticsdb.UnitTools

	return t.Pick(t.ModelKey, t.ProviderID, t.ToolName, t.Cause, t.Calls.Sum()).
		Where(toolScope(t, filter)...).
		Group(t.ModelKey, t.ProviderID, t.ToolName, t.Cause)
}

func toolKey(t *aianalyticsdb.UnitToolsTable, compareBy aianalytics.CompareBy) keel.StringCol {
	if compareBy == aianalytics.CompareByProvider {
		return t.ProviderID
	}

	return t.ModelKey
}

func taskCellsQuery(filter *aianalyticsports.Filter) *keel.Query[*aianalytics.WorkUnit] {
	w := aianalyticsdb.WorkUnits
	keys := append(deploymentColumns(w), w.Task)

	return w.Pick(append(append([]keel.Expr{}, keys...), keel.Count(),
		keel.Count().Filter(completedUnit(w)))...).
		Where(unitWindow(w, filter)...).
		Group(keys...)
}

func decisionsQuery(filter *aianalyticsports.Filter) *keel.Query[*agentdecision.AgentDecision] {
	d := agentdecisiondb.AgentDecisions
	p := agentdb.AgentProposals
	w := aianalyticsdb.WorkUnits
	keys := deploymentColumns(w)

	return d.Join(p, p.ID.EqCol(d.ProposalID)).
		Join(w, w.DecisionRunID.EqCol(p.RunID)).
		Pick(append(append([]keel.Expr{}, keys...),
			keel.Count().Filter(d.Decision.Eq(agentdecision.DecisionAccepted)),
			keel.Count().Filter(d.Decision.Eq(agentdecision.DecisionModified)),
			keel.Count().Filter(d.Decision.Eq(agentdecision.DecisionRejected)),
		)...).
		Where(append(unitScope(w, filter),
			d.CreatedAt.Gte(filter.Since),
			d.ProposalID.IsNotNull(),
			d.UndoneAt.IsNull(),
		)...).
		Group(keys...)
}

func ratingsQuery(filter *aianalyticsports.Filter) *keel.Query[*aifeedback.Feedback] {
	f := aifeedbackdb.Feedbacks
	w := aianalyticsdb.WorkUnits
	keys := deploymentColumns(w)

	return f.Join(w, w.ID.EqCol(f.TurnID.Coalesce(f.RunID))).
		Pick(append(append([]keel.Expr{}, keys...),
			keel.Count(),
			keel.Count().Filter(f.Rating.Gt(0)),
		)...).
		Where(append(unitScope(w, filter), f.CreatedAt.Gte(filter.Since))...).
		Group(keys...)
}

func suggestionsQuery(
	req *aianalyticsports.SuggestionRequest,
) *keel.Query[*agentsuggestion.Suggestion] {
	s := agentsuggestiondb.Suggestions
	d := agentdefinitiondb.Definitions

	return s.LeftJoin(d, d.ID.EqCol(s.SuggestedAgentID)).
		Pick(
			s.SuggestedAgentID,
			keel.StringOf(d.Name.Max()).Coalesce(""),
			keel.Count().Filter(s.Outcome.Eq(agentsuggestion.OutcomeSwitched)),
			keel.Count().Filter(s.Outcome.Eq(agentsuggestion.OutcomeSentThrough)),
			keel.Count().Filter(s.Outcome.Eq(agentsuggestion.OutcomeAutoSwitched)),
			keel.Count().Filter(s.Outcome.Eq(agentsuggestion.OutcomeKept)),
			keel.Count().Filter(s.Outcome.In(
				agentsuggestion.OutcomePassedOver,
				agentsuggestion.OutcomeShown,
			)),
		).
		Where(
			s.CreatedAt.Gte(req.Filter.Since),
			keel.When(
				req.Filter.AgentDefinitionID.IsNotNil(),
				s.SuggestedAgentID.Eq(req.Filter.AgentDefinitionID),
			),
			keel.Or(
				s.Outcome.Ne(agentsuggestion.OutcomeShown),
				s.CreatedAt.Lt(req.PendingSince),
			),
		).
		Group(s.SuggestedAgentID)
}

func providerCallsQuery(filter *aianalyticsports.Filter) *keel.Query[*aianalytics.ProviderHour] {
	h := aianalyticsdb.ProviderHours

	return h.Pick(h.ModelKey, h.ProviderID, h.ErrorClass,
		h.Calls.Sum(), h.FellToNext.Sum(), h.Hour.Max()).
		Where(hourScope(h, filter)...).
		Group(h.ModelKey, h.ProviderID, h.ErrorClass)
}

func firstTokenQuery(filter *aianalyticsports.Filter) *keel.Query[*aianalytics.ProviderHour] {
	h := aianalyticsdb.ProviderHours
	hist := keel.Int64ArrayOf(h.FirstTokenHist)
	b := keel.Rows("b", []string{"v", "idx"},
		hist.Unnest(),
		keel.GenerateSeries(keel.ConstInt64(1), keel.Int64Of(hist.Len())),
	)
	bucket := keel.Int64Of(b.Col("idx")).Sub(keel.ConstInt64(1))

	return h.JoinLateral(b).
		Pick(h.ModelKey, h.ProviderID, bucket, keel.Sum(b.Col("v"))).
		Where(append(hourScope(h, filter),
			h.ErrorClass.Eq(aianalytics.ProviderErrorNone),
			b.Col("v").Gt(0),
		)...).
		Group(h.ModelKey, h.ProviderID, bucket)
}

func providerHoursQuery(filter *aianalyticsports.Filter) *keel.Query[*aianalytics.ProviderHour] {
	h := aianalyticsdb.ProviderHours

	return h.Pick(
		h.ProviderID,
		h.Hour,
		h.Calls.Sum(),
		keel.NumericOf(h.Calls.Sum().Filter(h.ErrorClass.In(aianalytics.AllProviderErrors()...))).
			Coalesce(0),
		keel.NumericOf(h.Calls.Sum().Filter(h.ErrorClass.Eq(aianalytics.ProviderErrorAccessDenied))).
			Coalesce(0),
	).
		Where(append(hourScope(h, filter), h.ProviderID.Ne(""))...).
		Group(h.ProviderID, h.Hour).
		Order(h.ProviderID.Asc(), h.Hour.Asc())
}

func providerVersionsQuery(
	filter *aianalyticsports.Filter,
) *keel.Query[*settingversion.SettingVersion] {
	v := settingversiondb.SettingVersions
	changed := settingversiondb.SettingVersions.As("changed")

	return v.Pick(
		v.SubjectID,
		v.Version,
		v.Snapshot.Text("name").Coalesce(""),
		v.Snapshot.Text("model").Coalesce(""),
		keel.CastBool(v.Snapshot.Text("enabled")).Coalesce(false),
		v.CreatedAt,
	).
		Where(
			v.Kind.Eq(settingversion.KindAIProvider),
			v.SubjectID.InQuery(changed.Pick(changed.SubjectID).Where(
				changed.Kind.Eq(settingversion.KindAIProvider),
				changed.CreatedAt.Gte(filter.Since),
			)),
		).
		Order(v.SubjectID.Asc(), v.Version.Asc())
}

func toolCausesQuery(req *aianalyticsports.ToolFailureRequest) *keel.Query[*aianalytics.UnitTool] {
	t := aianalyticsdb.UnitTools

	return t.Pick(t.Cause, t.ProviderID, t.Calls.Sum()).
		Where(append(toolScope(t, &req.Filter),
			t.ToolName.Eq(req.ToolName),
			toolKey(t, req.CompareBy).Eq(req.SeriesKey),
		)...).
		Group(t.Cause, t.ProviderID)
}

func toolExamplesQuery(req *aianalyticsports.ToolFailureRequest) *keel.Query[*keel.Row] {
	t := aianalyticsdb.UnitTools
	w := aianalyticsdb.WorkUnits
	s := agentrunstepdb.AgentRunSteps
	latest := t.FinishedAt.Max()
	candidates := keel.CTE("candidates", []string{colUnitID, "finished_at"}, t.
		Pick(t.UnitID, latest).
		Where(append(toolScope(t, &req.Filter),
			t.ToolName.Eq(req.ToolName),
			toolKey(t, req.CompareBy).Eq(req.SeriesKey),
			t.Cause.Ne(aianalytics.Cause("")),
		)...).
		Group(t.UnitID).
		Order(latest.Desc()).
		Limit(req.Limit*toolExampleCandidatesPerExample))
	unitID := candidates.Col(colUnitID)
	outcome := s.Outcome

	return candidates.Query().With(candidates).
		Join(w, w.ID.EqCol(unitID)).
		Join(s, s.OwnerID.EqCol(unitID)).
		Pick(
			w.ID,
			w.Source,
			s.ToolName,
			outcome.Text("content").
				NullIf("").
				Coalesce(outcome.Text("reason"), "").
				Left(300),
			w.AgentDefinitionID.Coalesce(""),
			s.SettledAtMs.Div(keel.Int64(1000)).Coalesce(s.UpdatedAt),
		).
		Where(
			s.Kind.Eq(string(agentruntimeports.RunStepTool)),
			s.Status.Eq(string(agentruntimeports.RunStepFailed)),
			s.ToolName.Eq(req.ToolName),
		).
		Order(s.UpdatedAt.Desc(), s.ID.Desc()).
		Limit(req.Limit)
}

func paletteQuery(since int64) *keel.Query[*aianalytics.ProviderHour] {
	h := aianalyticsdb.ProviderHours

	return h.Pick(h.ModelKey, h.ProviderID).Where(h.Hour.Gte(floorHour(since))).Distinct()
}
