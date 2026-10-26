package aianalyticsrepository

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics/aianalyticsdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage/aiusagedb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const (
	callUnitPrefix  = "call_"
	lookbackSeconds = 3600
	secondsPerHour  = 3600
	msPerSecond     = 1000

	authFailurePattern = `(\m401\M|\m403\M|unauthori[sz]ed|forbidden|access denied|invalid api key|invalid x-api-key|authentication|permission denied|not authorized)`
	rateLimitPattern   = `(\m429\M|rate.?limit|too many requests|quota exceeded|throttl)`
)

var (
	timeoutPhrases = []string{"deadline exceeded", "timed out", "timeout"}

	callColumns = []string{
		"model_calls", "failed_calls", "failovers", "input_tokens", "output_tokens",
		"cache_read_tokens", "cache_write_tokens", "cost_usd", "advisor_calls",
		"advisor_prompt_tokens", "advisor_output_tokens", "advisor_cost_usd", "first_start_ms",
		"last_end_ms", "first_token_ms", "model", "provider_id", "task", "agent_definition_id",
		"answered", "last_call_at",
	}
	countedCallColumns = []string{
		"model_calls", "failed_calls", "failovers", "input_tokens", "output_tokens",
		"cache_read_tokens", "cache_write_tokens",
	}
	advisorCountColumns = []string{
		"advisor_calls", "advisor_prompt_tokens", "advisor_output_tokens",
	}
	stepColumns = []string{
		"owner_id", "tool_calls", "tool_failures_model", "tool_failures_tool", "last_end_ms",
	}
	hourKeyColumns = []string{"hour", "provider_id", "model", "task", "agent_definition_id"}

	unitSources = []aianalytics.Source{
		aianalytics.SourceAgentRun,
		aianalytics.SourceAssistantTurn,
	}
)

type unitColumn interface {
	keel.Column
	keel.Expr
	Excluded() keel.Expr
}

func trackedUnitColumns(w *aianalyticsdb.WorkUnitsTable) []unitColumn {
	return []unitColumn{
		w.Source, w.AgentDefinitionID, w.ProviderID, w.Model, w.Task, w.Outcome,
		w.DecisionRunID, w.StartedAtMs, w.FinishedAt, w.DurationMs, w.FirstTokenMs,
		w.ModelCalls, w.FailedCalls, w.Failovers, w.InputTokens, w.OutputTokens,
		w.CacheReadTokens, w.CacheWriteTokens, w.CostUSD, w.AdvisorCalls,
		w.AdvisorPromptTokens, w.AdvisorOutputTokens, w.AdvisorCostUSD, w.ToolCalls,
		w.ToolFailuresModel, w.ToolFailuresTool,
	}
}

func unitConflict() keel.Conflict {
	w := aianalyticsdb.WorkUnits
	tracked := trackedUnitColumns(w)
	updated := make([]keel.Column, 0, len(tracked)+1)
	stored := make([]any, 0, len(tracked))
	proposed := make([]any, 0, len(tracked))
	for _, col := range tracked {
		updated = append(updated, col)
		stored = append(stored, col)
		proposed = append(proposed, col.Excluded())
	}
	updated = append(updated, w.RefreshStamp)

	return keel.OnConflict(w.ID, w.BusinessUnitID, w.OrganizationID).
		Update(updated...).
		Set(w.ModelKey.ToExpr(keel.CaseString().
			When(w.Model.EqCol(w.Model.Excluded()), w.ModelKey).
			Else(keel.Text("")))).
		UpdateWhere(keel.Tuple(stored...).IsDistinctFrom(keel.Tuple(proposed...)))
}

func unitInsertColumns(w *aianalyticsdb.WorkUnitsTable, withDecisionRun bool) []keel.Column {
	cols := []keel.Column{w.ID, w.Source, w.AgentDefinitionID, w.ProviderID, w.Model, w.ModelKey,
		w.Task, w.Outcome}
	if withDecisionRun {
		cols = append(cols, w.DecisionRunID)
	}

	return append(cols,
		w.StartedAtMs, w.FinishedAt, w.DurationMs, w.FirstTokenMs, w.ModelCalls, w.FailedCalls,
		w.Failovers, w.InputTokens, w.OutputTokens, w.CacheReadTokens, w.CacheWriteTokens,
		w.CostUSD, w.AdvisorCalls, w.AdvisorPromptTokens, w.AdvisorOutputTokens,
		w.AdvisorCostUSD, w.ToolCalls, w.ToolFailuresModel, w.ToolFailuresTool, w.RefreshStamp,
	)
}

func stepCause(s *agentdb.AgentRunStepsTable) keel.StringExpr {
	cause := s.Outcome.Text("cause")
	verdict := s.Outcome.Text("verdict")
	reason := s.Outcome.Text("reason")
	lowered := s.Outcome.Text("reason").Coalesce("").Lower()
	timedOut := make([]keel.Pred, 0, len(timeoutPhrases))
	for _, phrase := range timeoutPhrases {
		timedOut = append(timedOut, lowered.Contains(phrase))
	}

	return keel.CaseString().
		When(s.Status.Ne(string(agentports.RunStepFailed)), keel.Text("")).
		When(
			cause.Coalesce("").In(sliceutils.Strings(aianalytics.AllCauses())...),
			cause,
		).
		When(
			keel.And(
				verdict.Eq(aianalytics.VerdictInvalid),
				reason.Eq(aianalytics.ReasonUnregistered),
			),
			keel.Text(string(aianalytics.CauseToolNotHeld)),
		).
		When(
			verdict.Eq(aianalytics.VerdictInvalid),
			keel.Text(string(aianalytics.CauseWrongArguments)),
		).
		When(
			verdict.Eq(aianalytics.VerdictDuplicate),
			keel.Text(string(aianalytics.CauseRepeatedCall)),
		).
		When(
			keel.And(verdict.Eq(aianalytics.VerdictDenied), reason.Eq(aianalytics.ReasonNotHeld)),
			keel.Text(string(aianalytics.CauseToolNotHeld)),
		).
		When(
			verdict.In(aianalytics.VerdictDenied, aianalytics.VerdictOverBudget),
			keel.Text(string(aianalytics.CauseBlockedByRule)),
		).
		When(keel.Or(timedOut...), keel.Text(string(aianalytics.CauseToolTimedOut))).
		Else(keel.Text(string(aianalytics.CauseToolError)))
}

func providerError(u *aiusagedb.AIUsageRecordsTable) keel.StringExpr {
	return keel.CaseString().
		When(u.Succeeded.IsTrue(), keel.Text(string(aianalytics.ProviderErrorNone))).
		When(
			u.ErrorClass.Eq(aiusage.ErrorClassCancelled),
			keel.Text(string(aianalytics.ProviderErrorCancelled)),
		).
		When(
			u.ErrorClass.Eq(aiusage.ErrorClassRefused),
			keel.Text(string(aianalytics.ProviderErrorDeclined)),
		).
		When(
			keel.Or(
				u.ErrorClass.Eq(aiusage.ErrorClassAuth),
				u.ErrorMessage.MatchFold(authFailurePattern),
			),
			keel.Text(string(aianalytics.ProviderErrorAccessDenied)),
		).
		When(
			keel.Or(
				u.ErrorClass.Eq(aiusage.ErrorClassRateLimited),
				u.ErrorMessage.MatchFold(rateLimitPattern),
			),
			keel.Text(string(aianalytics.ProviderErrorRateLimited)),
		).
		When(
			u.ErrorClass.Eq(aiusage.ErrorClassTimeout),
			keel.Text(string(aianalytics.ProviderErrorTimedOut)),
		).
		When(
			u.ErrorClass.Eq(aiusage.ErrorClassNetwork),
			keel.Text(string(aianalytics.ProviderErrorCouldNotConnect)),
		).
		Else(keel.Text(string(aianalytics.ProviderErrorServer)))
}

func callAggregates(u *aiusagedb.AIUsageRecordsTable) []keel.Expr {
	callEnd := u.RecordedAtMs.Coalesce(u.CreatedAt.Mul(keel.Int64(msPerSecond)))
	callStart := callEnd.Sub(u.LatencyMs)
	latestFirst := []keel.Order{u.Succeeded.Desc(), callEnd.Desc()}
	first := func(agg keel.Agg) keel.StringExpr { return keel.StringArrayOf(agg).At(1) }

	return []keel.Expr{
		keel.Count(),
		keel.Count().Filter(u.Succeeded.Eq(false)),
		keel.Count().Filter(u.Failover.IsTrue()),
		summed(keel.CaseInt64().
			When(u.ProviderKind.Eq(aiprovider.KindAnthropicMessages), u.InputTokens).
			Else(u.InputTokens.Sub(u.CacheReadTokens).
				Sub(u.CacheWriteTokens).
				Greatest(keel.ConstInt64(0)))),
		summed(u.OutputTokens),
		summed(u.CacheReadTokens),
		summed(u.CacheWriteTokens),
		u.CostUSD.Sum(),
		summed(u.AdvisorCalls),
		summed(u.AdvisorInputTokens.
			Add(u.AdvisorCacheReadTokens).
			Add(u.AdvisorCacheWriteTokens)),
		summed(u.AdvisorOutputTokens),
		u.AdvisorCostUSD.Sum(),
		keel.Min(callStart),
		keel.Max(callEnd),
		keel.Int64ArrayOf(keel.ArrayAgg(u.FirstTokenMs).
			OrderBy(callStart.Asc()).
			Filter(u.FirstTokenMs.IsNotNull())).At(1),
		first(keel.ArrayAgg(u.Model).OrderBy(latestFirst...)),
		first(keel.ArrayAgg(u.ProviderID).OrderBy(latestFirst...)),
		first(keel.ArrayAgg(u.Task).OrderBy(latestFirst...)),
		first(keel.ArrayAgg(u.AgentDefinitionID).
			OrderBy(callEnd.Asc()).
			Filter(u.AgentDefinitionID.IsNotNull())),
		keel.BoolOr(keel.BoolOf(u.Succeeded)),
		u.CreatedAt.Max(),
	}
}

type ownedSource struct {
	source    aianalytics.Source
	ownerKind agent.RunOwnerKind
	owners    func(req *unitRefresh) *keel.CTETable
	agent     func(owners, calls *keel.CTETable) keel.Expr
	outcome   func(owners, handed *keel.CTETable) keel.Expr
}

type unitRefresh struct {
	from  int64
	to    int64
	stamp int64
}

var ownerColumns = []string{
	"id", "decision_run_id", "provider_hint", "model_hint", "task_hint", "status", "at_limit",
	"started_ms", "finished_at",
}

func providerHint(fingerprint keel.JSONBCol) keel.StringExpr {
	return fingerprint.Text("providerId").NullIf("")
}

func modelHint(fingerprint keel.JSONBCol) keel.StringExpr {
	return fingerprint.Text("model").NullIf("")
}

func stoppedAtLimit(fingerprint keel.JSONBCol) keel.BoolExpr {
	return keel.CastBool(fingerprint.Text("stoppedAtLimit")).Coalesce(false)
}

func ownerStartedMs(startedAt, createdAt keel.Int64Col) keel.Int64Expr {
	return startedAt.NullIf(0).Coalesce(createdAt).Mul(keel.Int64(msPerSecond))
}

func runOwners(req *unitRefresh) *keel.CTETable {
	o := agentdb.AgentRuns

	return keel.CTE("owners", append(append([]string{}, ownerColumns...), "agent_definition_id"), o.
		Pick(
			o.ID,
			o.ID,
			providerHint(o.Fingerprint),
			modelHint(o.Fingerprint).Coalesce(o.ModelIdentifier.NullIf("")),
			keel.CaseString().
				When(o.AgentType.Eq(agent.TypeBillingException),
					keel.Text(string(aiprovider.TaskBillingDiagnosis))).
				When(o.AgentType.Eq(agent.TypeAssistantChat),
					keel.Text(string(aiprovider.TaskAssistantChat))).
				Else(keel.Text(string(aiprovider.TaskGeneral))),
			o.Status,
			stoppedAtLimit(o.Fingerprint),
			ownerStartedMs(o.StartedAt, o.CreatedAt),
			o.CompletedAt.Coalesce(o.UpdatedAt),
			o.AgentDefinitionID,
		).
		Where(
			o.UpdatedAt.Gte(req.from),
			o.UpdatedAt.Lt(req.to),
			o.Status.In(
				agent.RunStatusCompleted,
				agent.RunStatusShadowCompleted,
				agent.RunStatusAwaitingDecision,
				agent.RunStatusFailed,
			),
			o.TurnID.IsNull(),
		))
}

func turnOwners(req *unitRefresh) *keel.CTETable {
	o := conversationdb.AssistantTurns

	return keel.CTE("owners", ownerColumns, o.
		Pick(
			o.ID,
			o.RunID,
			providerHint(o.Fingerprint),
			modelHint(o.Fingerprint),
			keel.Text(string(aiprovider.TaskAssistantChat)),
			o.Status,
			stoppedAtLimit(o.Fingerprint),
			ownerStartedMs(o.StartedAt, o.CreatedAt),
			o.CompletedAt.Coalesce(o.UpdatedAt),
		).
		Where(
			o.UpdatedAt.Gte(req.from),
			o.UpdatedAt.Lt(req.to),
			o.Status.In(
				conversation.AssistantTurnStatusCompleted,
				conversation.AssistantTurnStatusRefused,
				conversation.AssistantTurnStatusStopped,
				conversation.AssistantTurnStatusFailed,
			),
			o.Origin.Ne(conversation.AssistantTurnOriginCompaction),
		))
}

var runSource = ownedSource{
	source:    aianalytics.SourceAgentRun,
	ownerKind: agent.RunOwnerAgentRun,
	owners:    runOwners,
	agent: func(owners, calls *keel.CTETable) keel.Expr {
		return owners.Col("agent_definition_id").Coalesce(calls.Col("agent_definition_id"))
	},
	outcome: func(owners, handed *keel.CTETable) keel.Expr {
		return keel.CaseString().
			When(owners.Col("status").Eq(string(agent.RunStatusFailed)),
				keel.Text(string(aianalytics.OutcomeFailed))).
			When(handed.Col("run_id").IsNotNull(), keel.Text(string(aianalytics.OutcomeHandedOff))).
			When(keel.BoolOf(owners.Col("at_limit")).IsTrue(),
				keel.Text(string(aianalytics.OutcomeStopped))).
			Else(keel.Text(string(aianalytics.OutcomeCompleted)))
	},
}

var turnSource = ownedSource{
	source:    aianalytics.SourceAssistantTurn,
	ownerKind: agent.RunOwnerAssistantTurn,
	owners:    turnOwners,
	agent: func(_, calls *keel.CTETable) keel.Expr {
		return calls.Col("agent_definition_id")
	},
	outcome: func(owners, handed *keel.CTETable) keel.Expr {
		return keel.CaseString().
			When(owners.Col("status").Eq(string(conversation.AssistantTurnStatusFailed)),
				keel.Text(string(aianalytics.OutcomeFailed))).
			When(owners.Col("status").Eq(string(conversation.AssistantTurnStatusStopped)),
				keel.Text(string(aianalytics.OutcomeStopped))).
			When(keel.BoolOf(owners.Col("at_limit")).IsTrue(),
				keel.Text(string(aianalytics.OutcomeStopped))).
			When(handed.Col("run_id").IsNotNull(), keel.Text(string(aianalytics.OutcomeHandedOff))).
			Else(keel.Text(string(aianalytics.OutcomeCompleted)))
	},
}

func zeroed(col keel.AnyCol) keel.NumericExpr {
	return keel.NumericOf(col).Coalesce(0)
}

func ownedUnitsSource(src *ownedSource, req *unitRefresh) *keel.Query[*keel.Row] {
	owners := src.owners(req)

	u := aiusagedb.AIUsageRecords
	calls := keel.CTE("calls", append([]string{"owner_id"}, callColumns...), u.
		Pick(append([]keel.Expr{u.OwnerID}, callAggregates(u)...)...).
		Where(u.OwnerKind.Eq(src.ownerKind), u.OwnerID.InQuery(owners.Query().Pick(owners.Col("id")))).
		Group(u.OwnerID))

	s := agentdb.AgentRunSteps
	stepCauses := keel.Rows("c", []string{"cause"}, stepCause(s))
	cause := stepCauses.Col("cause")
	modelCauses := sliceutils.Strings(aianalytics.ModelCauses())
	steps := keel.CTE("steps", stepColumns, s.
		JoinLateral(stepCauses).
		Pick(
			s.OwnerID,
			keel.Count(),
			keel.Count().Filter(cause.In(modelCauses)),
			keel.Count().Filter(keel.And(cause.Ne(""), cause.NotIn(modelCauses))),
			keel.Max(s.SettledAtMs.Coalesce(
				s.StartedAtMs.Coalesce(s.UpdatedAt.Mul(keel.Int64(msPerSecond))),
			)),
		).
		Where(
			s.OwnerKind.Eq(string(src.ownerKind)),
			s.OwnerID.InQuery(owners.Query().Pick(owners.Col("id"))),
			s.Kind.Eq(string(agentports.RunStepTool)),
			s.Status.Ne(string(agentports.RunStepStarted)),
		).
		Group(s.OwnerID))

	e := agentdb.AgentExceptions
	decisionRun := owners.Col("decision_run_id")
	handed := keel.CTE("handed", []string{"run_id"}, e.
		Pick(e.RunID).
		Where(e.RunID.InQuery(owners.Query().Pick(decisionRun).Where(decisionRun.IsNotNull()))).
		Distinct())

	callEnd := calls.Col("last_end_ms")
	stepEnd := steps.Col("last_end_ms")
	startedAt := calls.Col("first_start_ms").Coalesce(owners.Col("started_ms"))
	finishedAt := owners.Col("finished_at")
	model := keel.StringOf(calls.Col("model")).Coalesce(owners.Col("model_hint"))
	ownerEnd := keel.CaseInt64().
		When(
			keel.And(callEnd.IsNull(), stepEnd.IsNull()),
			keel.Int64Of(finishedAt).Mul(keel.Int64(msPerSecond)),
		).
		Else(keel.ConstInt64(0))
	duration := keel.ConstInt64(0).Greatest(
		keel.Int64Of(zeroed(callEnd)).
			Greatest(keel.Int64Of(zeroed(stepEnd)), ownerEnd).
			Sub(keel.Int64Of(startedAt)),
	)

	picks := []keel.Expr{
		owners.Col("id"),
		keel.Text(string(src.source)),
		src.agent(owners, calls),
		calls.Col("provider_id").Coalesce(owners.Col("provider_hint")),
		model,
		keel.Text(""),
		calls.Col("task").Coalesce(owners.Col("task_hint")),
		src.outcome(owners, handed),
		decisionRun,
		startedAt,
		finishedAt,
		duration,
		calls.Col("first_token_ms"),
	}
	for _, name := range countedCallColumns {
		picks = append(picks, zeroed(calls.Col(name)))
	}
	picks = append(picks, calls.Col("cost_usd"))
	for _, name := range advisorCountColumns {
		picks = append(picks, zeroed(calls.Col(name)))
	}
	picks = append(picks,
		calls.Col("advisor_cost_usd"),
		zeroed(steps.Col("tool_calls")),
		zeroed(steps.Col("tool_failures_model")),
		zeroed(steps.Col("tool_failures_tool")),
		keel.Int64(req.stamp),
	)

	return owners.Query().
		With(owners, calls, steps, handed).
		LeftJoin(calls, calls.Col("owner_id").EqCol(owners.Col("id"))).
		LeftJoin(steps, steps.Col("owner_id").EqCol(owners.Col("id"))).
		LeftJoin(handed, handed.Col("run_id").EqCol(decisionRun)).
		Pick(picks...).
		Where(model.IsNotNull())
}

func callUnitKey(u *aiusagedb.AIUsageRecordsTable) keel.StringExpr {
	return keel.CaseString().
		When(u.TraceID.IsNotNull(), keel.Text(callUnitPrefix).Cat(
			u.TraceID.Cat(
				keel.Text(":"),
				u.Task,
				keel.Text(":"),
				u.SubjectID.Coalesce(""),
			).MD5(),
		)).
		Else(u.ID)
}

func standaloneCalls(u *aiusagedb.AIUsageRecordsTable) []keel.Pred {
	return []keel.Pred{
		u.OwnerID.IsNull(),
		u.RunID.IsNull(),
		u.ThreadID.IsNull(),
		u.Surface.NotIn(
			aiusage.SurfaceEvaluation,
			aiusage.SurfaceIndexing,
			aiusage.SurfaceRetrieval,
		),
		u.Feature.Coalesce("").Ne(string(aiusage.FeatureAgentEvaluation)),
	}
}

func callUnitsSource(req *unitRefresh) *keel.Query[*keel.Row] {
	u := aiusagedb.AIUsageRecords
	key := callUnitKey(u)
	touched := keel.CTE("touched", []string{colUnitID}, u.
		Pick(key).
		Where(append(standaloneCalls(u), u.CreatedAt.Gte(req.from), u.CreatedAt.Lt(req.to))...).
		Distinct())
	touchedID := touched.Col(colUnitID)
	calls := keel.CTE("calls", append([]string{colUnitID}, callColumns...), u.
		Pick(append([]keel.Expr{key}, callAggregates(u)...)...).
		Where(append(standaloneCalls(u),
			u.CreatedAt.Gte(req.from-lookbackSeconds),
			u.CreatedAt.Lt(req.to),
			keel.Exists(touched.Query().Pick(touchedID).Where(touchedID.EqCol(key))),
		)...).
		Group(key))
	col := calls.Col

	picks := []keel.Expr{
		col(colUnitID),
		keel.Text(string(aianalytics.SourceCall)),
		col("agent_definition_id"),
		col("provider_id"),
		col("model"),
		keel.Text(""),
		col("task"),
		keel.CaseString().
			When(
				keel.BoolOf(col("answered")).IsTrue(),
				keel.Text(string(aianalytics.OutcomeCompleted)),
			).
			Else(keel.Text(string(aianalytics.OutcomeFailed))),
		col("first_start_ms"),
		col("last_call_at"),
		keel.ConstInt64(0).Greatest(
			keel.Int64Of(col("last_end_ms")).Sub(keel.Int64Of(col("first_start_ms"))),
		),
		col("first_token_ms"),
	}
	for _, name := range countedCallColumns {
		picks = append(picks, col(name))
	}
	picks = append(picks, col("cost_usd"))
	for _, name := range advisorCountColumns {
		picks = append(picks, col(name))
	}
	picks = append(picks,
		col("advisor_cost_usd"),
		keel.ConstInt64(0),
		keel.ConstInt64(0),
		keel.ConstInt64(0),
		keel.Int64(req.stamp),
	)

	return calls.Query().
		With(touched, calls).
		Pick(picks...).
		Where(col("model").IsNotNull())
}

func clearUnitToolsQuery(stamp int64) *keel.Query[*aianalytics.UnitTool] {
	t := aianalyticsdb.UnitTools
	w := aianalyticsdb.WorkUnits

	return t.Join(w, w.ID.EqCol(t.UnitID)).
		Where(w.RefreshStamp.Eq(stamp), w.Source.In(unitSources...))
}

func unitToolsSource(stamp int64) *keel.Query[*aianalytics.WorkUnit] {
	w := aianalyticsdb.WorkUnits
	s := agentdb.AgentRunSteps
	stepCauses := keel.Rows("c", []string{"cause"}, stepCause(s))
	cause := stepCauses.Col("cause")

	return w.Join(s, s.OwnerID.EqCol(w.ID)).
		JoinLateral(stepCauses).
		Pick(
			s.OwnerID,
			s.ToolName,
			cause,
			w.FinishedAt,
			w.ModelKey,
			w.ProviderID.Coalesce(""),
			w.AgentDefinitionID.Coalesce(""),
			w.Task,
			keel.Count(),
		).
		Where(
			w.RefreshStamp.Eq(stamp),
			w.Source.In(unitSources...),
			s.Kind.Eq(string(agentports.RunStepTool)),
			s.Status.Ne(string(agentports.RunStepStarted)),
			s.ToolName.Ne(""),
		).
		Group(s.OwnerID, s.ToolName, cause, w.FinishedAt, w.ModelKey, w.ProviderID,
			w.AgentDefinitionID, w.Task)
}

func unitToolColumns(t *aianalyticsdb.UnitToolsTable) []keel.Column {
	return []keel.Column{t.UnitID, t.ToolName, t.Cause, t.FinishedAt, t.ModelKey, t.ProviderID,
		t.AgentDefinitionID, t.Task, t.Calls}
}

func clearProviderHoursQuery(fromHour, toHour int64) *keel.Query[*aianalytics.ProviderHour] {
	h := aianalyticsdb.ProviderHours

	return h.Where(h.Hour.Gte(fromHour), h.Hour.Lt(toHour))
}

func firstTokenBucket(firstTokenMs keel.Expr) keel.Int64Expr {
	ms := keel.Int64Of(firstTokenMs)
	octaves := keel.CastFloat64(ms).Div(keel.ConstFloat64(aianalytics.HistogramBaseMs)).Ln()
	index := keel.ConstFloat64(aianalytics.HistogramStepsPerOctet).
		Mul(octaves).
		Div(keel.ConstFloat64(2).Ln()).
		Floor()

	return keel.CaseInt64().
		When(ms.LtExpr(keel.ConstInt64(aianalytics.HistogramBaseMs)), keel.ConstInt64(0)).
		Else(keel.ConstInt64(aianalytics.HistogramBuckets - 1).
			Least(keel.CastInt64(index).Add(keel.ConstInt64(1))))
}

func cteColumns(cte *keel.CTETable, names ...string) []keel.Expr {
	out := make([]keel.Expr, 0, len(names))
	for _, name := range names {
		out = append(out, cte.Col(name))
	}

	return out
}

func sameHourKey(left, right *keel.CTETable) []keel.Pred {
	out := make([]keel.Pred, 0, len(hourKeyColumns))
	for _, name := range hourKeyColumns {
		out = append(out, left.Col(name).EqCol(right.Col(name)))
	}

	return out
}

func providerHoursSource(fromHour, toHour int64) *keel.Query[*keel.Row] {
	u := aiusagedb.AIUsageRecords
	calls := keel.CTE("calls", append(append([]string{}, hourKeyColumns...),
		"error_class", "succeeded", "first_token_ms", "trace_id"), u.
		Pick(
			u.CreatedAt.Div(keel.Int64(secondsPerHour)).Mul(keel.Int64(secondsPerHour)),
			u.ProviderID.Coalesce(""),
			u.Model,
			u.Task,
			u.AgentDefinitionID.Coalesce(""),
			providerError(u),
			u.Succeeded,
			u.FirstTokenMs,
			u.TraceID,
		).
		Where(
			u.CreatedAt.Gte(fromHour),
			u.CreatedAt.Lt(toHour),
			u.Surface.NotIn(aiusage.SurfaceIndexing, aiusage.SurfaceRetrieval),
		))
	call := calls.Col

	rescued := keel.CTE("rescued", []string{"trace_id", "providers"}, calls.Query().
		Pick(call("trace_id"), keel.ArrayAgg(call("provider_id")).Distinct()).
		Where(keel.BoolOf(call("succeeded")).IsTrue(), call("trace_id").IsNotNull()).
		Group(call("trace_id")))

	groupKeys := cteColumns(calls, append(append([]string{}, hourKeyColumns...), "error_class")...)
	grouped := keel.CTE("grouped", append(append([]string{}, hourKeyColumns...),
		"error_class", "calls", "fell_to_next"), calls.Query().
		LeftJoin(rescued, rescued.Col("trace_id").EqCol(call("trace_id"))).
		Pick(append(append([]keel.Expr{}, groupKeys...),
			keel.Count(),
			keel.Count().Filter(keel.And(
				call("succeeded").Eq(false),
				rescued.Col("providers").IsNotNull(),
				keel.Not(keel.StringArrayOf(rescued.Col("providers")).
					ContainedByExpr(keel.ArrayOfString(keel.StringOf(call("provider_id"))))),
			)),
		)...).
		Group(groupKeys...))

	hourKeys := cteColumns(calls, hourKeyColumns...)
	bucket := firstTokenBucket(call("first_token_ms"))
	hist := keel.CTE("hist", append(append([]string{}, hourKeyColumns...), "bucket", "n"),
		calls.Query().
			Pick(append(append([]keel.Expr{}, hourKeys...), bucket, keel.Count())...).
			Where(keel.BoolOf(call("succeeded")).IsTrue(), call("first_token_ms").IsNotNull()).
			Group(append(append([]keel.Expr{}, hourKeys...), bucket)...))

	histKeys := keel.CTE("hist_keys", hourKeyColumns, hist.Query().
		Pick(cteColumns(hist, hourKeyColumns...)...).
		Distinct())
	series := keel.Rows("b", []string{"i"}, keel.GenerateSeries(
		keel.ConstInt64(0),
		keel.ConstInt64(aianalytics.HistogramBuckets-1),
	))
	keyColumns := cteColumns(histKeys, hourKeyColumns...)
	arrays := keel.CTE("arrays", append(append([]string{}, hourKeyColumns...), "first_token_hist"),
		histKeys.Query().
			JoinLateral(series).
			LeftJoin(hist, append(sameHourKey(hist, histKeys),
				hist.Col("bucket").EqCol(series.Col("i")))...).
			Pick(append(append([]keel.Expr{}, keyColumns...),
				keel.ArrayAgg(hist.Col("n").Coalesce(0)).
					OrderBy(series.Col("i").Asc()),
			)...).
			Group(keyColumns...))

	group := grouped.Col
	return grouped.Query().
		With(calls, rescued, grouped, hist, histKeys, arrays).
		LeftJoin(arrays, sameHourKey(arrays, grouped)...).
		Pick(
			group("hour"),
			group("provider_id"),
			group("model"),
			keel.Text(""),
			group("task"),
			group("agent_definition_id"),
			group("error_class"),
			group("calls"),
			group("fell_to_next"),
			keel.DynCase().
				When(
					group("error_class").Eq(string(aianalytics.ProviderErrorNone)),
					keel.DynOf(arrays.Col("first_token_hist")).Coalesce(keel.DynConst("{}")),
				).
				Else(keel.DynConst("{}")),
		)
}

func providerHourColumns(h *aianalyticsdb.ProviderHoursTable) []keel.Column {
	return []keel.Column{h.Hour, h.ProviderID, h.Model, h.ModelKey, h.Task, h.AgentDefinitionID,
		h.ErrorClass, h.Calls, h.FellToNext, h.FirstTokenHist}
}
