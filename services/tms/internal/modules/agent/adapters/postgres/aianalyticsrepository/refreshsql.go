package aianalyticsrepository

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

const (
	callUnitPrefix  = "call_"
	lookbackSeconds = 3600
)

func quoted[T ~string](values ...T) string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, "'"+strings.ReplaceAll(string(value), "'", "''")+"'")
	}

	return strings.Join(out, ", ")
}

var (
	modelCauseList = quoted(aianalytics.ModelCauses()...)
	causeList      = quoted(aianalytics.AllCauses()...)

	causeSQL = fmt.Sprintf(`CASE
		WHEN s.status <> '%[1]s' THEN ''
		WHEN COALESCE(s.outcome->>'cause', '') IN (%[2]s) THEN s.outcome->>'cause'
		WHEN s.outcome->>'verdict' = '%[3]s' AND s.outcome->>'reason' = %[4]s THEN '%[5]s'
		WHEN s.outcome->>'verdict' = '%[3]s' THEN '%[6]s'
		WHEN s.outcome->>'verdict' = '%[7]s' THEN '%[8]s'
		WHEN s.outcome->>'verdict' = '%[9]s' AND s.outcome->>'reason' = %[10]s THEN '%[5]s'
		WHEN s.outcome->>'verdict' IN ('%[9]s', '%[11]s') THEN '%[12]s'
		WHEN LOWER(COALESCE(s.outcome->>'reason', '')) ~ '(deadline exceeded|timed out|timeout)' THEN '%[13]s'
		ELSE '%[14]s'
	END`,
		agentports.RunStepFailed,
		causeList,
		aianalytics.VerdictInvalid,
		quoted(aianalytics.ReasonUnregistered),
		aianalytics.CauseToolNotHeld,
		aianalytics.CauseWrongArguments,
		aianalytics.VerdictDuplicate,
		aianalytics.CauseRepeatedCall,
		aianalytics.VerdictDenied,
		quoted(aianalytics.ReasonNotHeld),
		aianalytics.VerdictOverBudget,
		aianalytics.CauseBlockedByRule,
		aianalytics.CauseToolTimedOut,
		aianalytics.CauseToolError,
	)

	standaloneSurfaces = quoted(
		aiusage.SurfaceEvaluation,
		aiusage.SurfaceIndexing,
		aiusage.SurfaceRetrieval,
	)

	nonCompletionSurfaces = quoted(
		aiusage.SurfaceIndexing,
		aiusage.SurfaceRetrieval,
	)

	callEndMs = "COALESCE(u.recorded_at_ms, u.created_at * 1000)"

	bucketSQL = fmt.Sprintf(
		"CASE WHEN c.first_token_ms < %[1]d THEN 0 "+
			"ELSE LEAST(%[2]d, FLOOR(%[3]d * LN(c.first_token_ms::float8 / %[1]d) / LN(2))::int + 1) END",
		aianalytics.HistogramBaseMs,
		aianalytics.HistogramBuckets-1,
		aianalytics.HistogramStepsPerOctet,
	)

	authFailureSQL = `u.error_message ~* '(\m401\M|\m403\M|unauthori[sz]ed|forbidden|access denied|invalid api key|invalid x-api-key|authentication|permission denied|not authorized)'`

	rateLimitSQL = `u.error_message ~* '(\m429\M|rate.?limit|too many requests|quota exceeded|throttl)'`

	providerErrorSQL = fmt.Sprintf(`CASE
				WHEN u.succeeded THEN '%[1]s'
				WHEN u.error_class = '%[2]s' THEN '%[3]s'
				WHEN u.error_class = '%[4]s' THEN '%[5]s'
				WHEN u.error_class = '%[6]s' OR %[7]s THEN '%[8]s'
				WHEN u.error_class = '%[9]s' OR %[10]s THEN '%[11]s'
				WHEN u.error_class = '%[12]s' THEN '%[13]s'
				WHEN u.error_class = '%[14]s' THEN '%[15]s'
				ELSE '%[16]s'
			END`,
		aianalytics.ProviderErrorNone,
		aiusage.ErrorClassCancelled, aianalytics.ProviderErrorCancelled,
		aiusage.ErrorClassRefused, aianalytics.ProviderErrorDeclined,
		aiusage.ErrorClassAuth, authFailureSQL, aianalytics.ProviderErrorAccessDenied,
		aiusage.ErrorClassRateLimited, rateLimitSQL, aianalytics.ProviderErrorRateLimited,
		aiusage.ErrorClassTimeout, aianalytics.ProviderErrorTimedOut,
		aiusage.ErrorClassNetwork, aianalytics.ProviderErrorCouldNotConnect,
		aianalytics.ProviderErrorServer,
	)
)

type ownedSource struct {
	source        aianalytics.Source
	table         string
	ownerKind     agent.RunOwnerKind
	statuses      string
	extraWhere    string
	agentColumn   string
	decisionRun   string
	taskFallback  string
	outcomeSQL    string
	finishedAtSQL string
	startedAtSQL  string
	modelFallback string
}

func callAggregates() string {
	return fmt.Sprintf(`COUNT(*) AS model_calls,
			COUNT(*) FILTER (WHERE NOT u.succeeded) AS failed_calls,
			COUNT(*) FILTER (WHERE u.failover) AS failovers,
			COALESCE(SUM(CASE WHEN u.provider_kind = '%[2]s' THEN u.input_tokens
				ELSE GREATEST(u.input_tokens - u.cache_read_tokens - u.cache_write_tokens, 0)
				END), 0) AS input_tokens,
			COALESCE(SUM(u.output_tokens), 0) AS output_tokens,
			COALESCE(SUM(u.cache_read_tokens), 0) AS cache_read_tokens,
			COALESCE(SUM(u.cache_write_tokens), 0) AS cache_write_tokens,
			SUM(u.cost_usd) AS cost_usd,
			COALESCE(SUM(u.advisor_calls), 0) AS advisor_calls,
			COALESCE(SUM(u.advisor_input_tokens + u.advisor_cache_read_tokens +
				u.advisor_cache_write_tokens), 0) AS advisor_prompt_tokens,
			COALESCE(SUM(u.advisor_output_tokens), 0) AS advisor_output_tokens,
			SUM(u.advisor_cost_usd) AS advisor_cost_usd,
			MIN(%[1]s - u.latency_ms) AS first_start_ms,
			MAX(%[1]s) AS last_end_ms,
			(ARRAY_AGG(u.first_token_ms ORDER BY %[1]s - u.latency_ms)
				FILTER (WHERE u.first_token_ms IS NOT NULL))[1] AS first_token_ms,
			(ARRAY_AGG(u.model ORDER BY u.succeeded DESC, %[1]s DESC))[1] AS model,
			(ARRAY_AGG(u.provider_id ORDER BY u.succeeded DESC, %[1]s DESC))[1] AS provider_id,
			(ARRAY_AGG(u.task ORDER BY u.succeeded DESC, %[1]s DESC))[1] AS task,
			(ARRAY_AGG(u.agent_definition_id ORDER BY %[1]s)
				FILTER (WHERE u.agent_definition_id IS NOT NULL))[1] AS agent_definition_id,
			BOOL_OR(u.succeeded) AS answered,
			MAX(u.created_at) AS last_call_at`, callEndMs, aiprovider.KindAnthropicMessages)
}

const unitColumns = `id, business_unit_id, organization_id, source, agent_definition_id, provider_id,
	model, model_key, task, outcome, decision_run_id, started_at_ms, finished_at, duration_ms,
	first_token_ms, model_calls, failed_calls, failovers, input_tokens, output_tokens,
	cache_read_tokens, cache_write_tokens, cost_usd, advisor_calls, advisor_prompt_tokens,
	advisor_output_tokens, advisor_cost_usd, tool_calls, tool_failures_model, tool_failures_tool,
	refresh_stamp`

const unitConflict = `ON CONFLICT (id, business_unit_id, organization_id) DO UPDATE SET
	source = EXCLUDED.source,
	agent_definition_id = EXCLUDED.agent_definition_id,
	provider_id = EXCLUDED.provider_id,
	model = EXCLUDED.model,
	model_key = CASE WHEN ai_work_units.model = EXCLUDED.model THEN ai_work_units.model_key ELSE '' END,
	task = EXCLUDED.task,
	outcome = EXCLUDED.outcome,
	decision_run_id = EXCLUDED.decision_run_id,
	started_at_ms = EXCLUDED.started_at_ms,
	finished_at = EXCLUDED.finished_at,
	duration_ms = EXCLUDED.duration_ms,
	first_token_ms = EXCLUDED.first_token_ms,
	model_calls = EXCLUDED.model_calls,
	failed_calls = EXCLUDED.failed_calls,
	failovers = EXCLUDED.failovers,
	input_tokens = EXCLUDED.input_tokens,
	output_tokens = EXCLUDED.output_tokens,
	cache_read_tokens = EXCLUDED.cache_read_tokens,
	cache_write_tokens = EXCLUDED.cache_write_tokens,
	cost_usd = EXCLUDED.cost_usd,
	advisor_calls = EXCLUDED.advisor_calls,
	advisor_prompt_tokens = EXCLUDED.advisor_prompt_tokens,
	advisor_output_tokens = EXCLUDED.advisor_output_tokens,
	advisor_cost_usd = EXCLUDED.advisor_cost_usd,
	tool_calls = EXCLUDED.tool_calls,
	tool_failures_model = EXCLUDED.tool_failures_model,
	tool_failures_tool = EXCLUDED.tool_failures_tool,
	refresh_stamp = EXCLUDED.refresh_stamp
	WHERE (
		ai_work_units.source, ai_work_units.agent_definition_id, ai_work_units.provider_id,
		ai_work_units.model, ai_work_units.task, ai_work_units.outcome,
		ai_work_units.decision_run_id, ai_work_units.started_at_ms, ai_work_units.finished_at,
		ai_work_units.duration_ms, ai_work_units.first_token_ms, ai_work_units.model_calls,
		ai_work_units.failed_calls, ai_work_units.failovers, ai_work_units.input_tokens,
		ai_work_units.output_tokens, ai_work_units.cache_read_tokens,
		ai_work_units.cache_write_tokens, ai_work_units.cost_usd, ai_work_units.advisor_calls,
		ai_work_units.advisor_prompt_tokens, ai_work_units.advisor_output_tokens,
		ai_work_units.advisor_cost_usd, ai_work_units.tool_calls,
		ai_work_units.tool_failures_model, ai_work_units.tool_failures_tool
	) IS DISTINCT FROM (
		EXCLUDED.source, EXCLUDED.agent_definition_id, EXCLUDED.provider_id, EXCLUDED.model,
		EXCLUDED.task, EXCLUDED.outcome, EXCLUDED.decision_run_id, EXCLUDED.started_at_ms,
		EXCLUDED.finished_at, EXCLUDED.duration_ms, EXCLUDED.first_token_ms,
		EXCLUDED.model_calls, EXCLUDED.failed_calls, EXCLUDED.failovers, EXCLUDED.input_tokens,
		EXCLUDED.output_tokens, EXCLUDED.cache_read_tokens, EXCLUDED.cache_write_tokens,
		EXCLUDED.cost_usd, EXCLUDED.advisor_calls, EXCLUDED.advisor_prompt_tokens,
		EXCLUDED.advisor_output_tokens, EXCLUDED.advisor_cost_usd, EXCLUDED.tool_calls,
		EXCLUDED.tool_failures_model, EXCLUDED.tool_failures_tool
	)`

func ownedUnitsSQL(src ownedSource) string {
	return fmt.Sprintf(`WITH owners AS (
		SELECT o.*
		FROM %[1]s AS o
		WHERE o.organization_id = ?org AND o.business_unit_id = ?bu
			AND o.updated_at >= ?from AND o.updated_at < ?to
			AND o.status IN (%[2]s)%[3]s
	),
	calls AS (
		SELECT u.owner_id, %[4]s
		FROM ai_usage_records AS u
		WHERE u.organization_id = ?org AND u.business_unit_id = ?bu
			AND u.owner_kind = '%[5]s' AND u.owner_id IN (SELECT id FROM owners)
		GROUP BY u.owner_id
	),
	steps AS (
		SELECT s.owner_id,
			COUNT(*) AS tool_calls,
			COUNT(*) FILTER (WHERE c.cause IN (%[6]s)) AS tool_failures_model,
			COUNT(*) FILTER (WHERE c.cause <> '' AND c.cause NOT IN (%[6]s)) AS tool_failures_tool,
			MAX(COALESCE(s.settled_at_ms, s.started_at_ms, s.updated_at * 1000)) AS last_end_ms
		FROM agent_run_steps AS s
		CROSS JOIN LATERAL (SELECT %[7]s AS cause) AS c
		WHERE s.organization_id = ?org AND s.business_unit_id = ?bu
			AND s.owner_kind = '%[5]s' AND s.owner_id IN (SELECT id FROM owners)
			AND s.kind = '%[8]s' AND s.status <> '%[9]s'
		GROUP BY s.owner_id
	),
	handed AS (
		SELECT DISTINCT e.run_id
		FROM agent_exceptions AS e
		WHERE e.organization_id = ?org AND e.business_unit_id = ?bu
			AND e.run_id IN (SELECT %[10]s FROM owners AS o WHERE %[10]s IS NOT NULL)
	),
	units AS (
		SELECT o.id,
			COALESCE(%[11]s, c.agent_definition_id) AS agent_definition_id,
			COALESCE(c.provider_id, NULLIF(o.fingerprint->>'providerId', '')) AS provider_id,
			COALESCE(c.model, NULLIF(o.fingerprint->>'model', '')%[12]s) AS model,
			COALESCE(c.task, %[13]s) AS task,
			%[14]s AS outcome,
			%[10]s AS decision_run_id,
			COALESCE(c.first_start_ms, %[15]s) AS started_at_ms,
			%[16]s AS finished_at,
			c.last_end_ms AS call_end_ms,
			st.last_end_ms AS step_end_ms,
			c.first_token_ms,
			COALESCE(c.model_calls, 0) AS model_calls,
			COALESCE(c.failed_calls, 0) AS failed_calls,
			COALESCE(c.failovers, 0) AS failovers,
			COALESCE(c.input_tokens, 0) AS input_tokens,
			COALESCE(c.output_tokens, 0) AS output_tokens,
			COALESCE(c.cache_read_tokens, 0) AS cache_read_tokens,
			COALESCE(c.cache_write_tokens, 0) AS cache_write_tokens,
			c.cost_usd,
			COALESCE(c.advisor_calls, 0) AS advisor_calls,
			COALESCE(c.advisor_prompt_tokens, 0) AS advisor_prompt_tokens,
			COALESCE(c.advisor_output_tokens, 0) AS advisor_output_tokens,
			c.advisor_cost_usd,
			COALESCE(st.tool_calls, 0) AS tool_calls,
			COALESCE(st.tool_failures_model, 0) AS tool_failures_model,
			COALESCE(st.tool_failures_tool, 0) AS tool_failures_tool
		FROM owners AS o
		LEFT JOIN calls AS c ON c.owner_id = o.id
		LEFT JOIN steps AS st ON st.owner_id = o.id
		LEFT JOIN handed AS h ON h.run_id = %[10]s
	)
	INSERT INTO ai_work_units (%[17]s)
	SELECT units.id, ?bu, ?org, '%[18]s', units.agent_definition_id, units.provider_id,
		units.model, '', units.task, units.outcome, units.decision_run_id, units.started_at_ms,
		units.finished_at,
		GREATEST(0, GREATEST(
			COALESCE(units.call_end_ms, 0),
			COALESCE(units.step_end_ms, 0),
			CASE WHEN units.call_end_ms IS NULL AND units.step_end_ms IS NULL
				THEN units.finished_at * 1000 ELSE 0 END
		) - units.started_at_ms),
		units.first_token_ms, units.model_calls, units.failed_calls, units.failovers,
		units.input_tokens, units.output_tokens, units.cache_read_tokens,
		units.cache_write_tokens, units.cost_usd, units.advisor_calls,
		units.advisor_prompt_tokens, units.advisor_output_tokens, units.advisor_cost_usd,
		units.tool_calls,
		units.tool_failures_model, units.tool_failures_tool, ?stamp
	FROM units
	WHERE units.model IS NOT NULL
	%[19]s`,
		src.table,
		src.statuses,
		src.extraWhere,
		callAggregates(),
		src.ownerKind,
		modelCauseList,
		causeSQL,
		agentports.RunStepTool,
		agentports.RunStepStarted,
		src.decisionRun,
		src.agentColumn,
		src.modelFallback,
		src.taskFallback,
		src.outcomeSQL,
		src.startedAtSQL,
		src.finishedAtSQL,
		unitColumns,
		src.source,
		unitConflict,
	)
}

var runUnitsSQL = ownedUnitsSQL(ownedSource{
	source:    aianalytics.SourceAgentRun,
	table:     "agent_runs",
	ownerKind: agent.RunOwnerAgentRun,
	statuses: quoted(
		agent.RunStatusCompleted,
		agent.RunStatusShadowCompleted,
		agent.RunStatusAwaitingDecision,
		agent.RunStatusFailed,
	),
	extraWhere:  "\n\t\t\tAND o.turn_id IS NULL",
	agentColumn: "o.agent_definition_id",
	decisionRun: "o.id",
	taskFallback: fmt.Sprintf("CASE o.agent_type WHEN %s THEN '%s' WHEN %s THEN '%s' ELSE '%s' END",
		quoted(agent.TypeBillingException), aiprovider.TaskBillingDiagnosis,
		quoted(agent.TypeAssistantChat), aiprovider.TaskAssistantChat,
		aiprovider.TaskGeneral,
	),
	outcomeSQL: fmt.Sprintf(`CASE
				WHEN o.status = %s THEN '%s'
				WHEN h.run_id IS NOT NULL THEN '%s'
				WHEN COALESCE((o.fingerprint->>'stoppedAtLimit')::boolean, false) THEN '%s'
				ELSE '%s'
			END`,
		quoted(agent.RunStatusFailed), aianalytics.OutcomeFailed,
		aianalytics.OutcomeHandedOff,
		aianalytics.OutcomeStopped,
		aianalytics.OutcomeCompleted,
	),
	startedAtSQL:  "COALESCE(NULLIF(o.started_at, 0), o.created_at) * 1000",
	finishedAtSQL: "COALESCE(o.completed_at, o.updated_at)",
	modelFallback: ", NULLIF(o.model_identifier, '')",
})

var turnUnitsSQL = ownedUnitsSQL(ownedSource{
	source:    aianalytics.SourceAssistantTurn,
	table:     "assistant_turns",
	ownerKind: agent.RunOwnerAssistantTurn,
	statuses: quoted(
		conversation.AssistantTurnStatusCompleted,
		conversation.AssistantTurnStatusRefused,
		conversation.AssistantTurnStatusStopped,
		conversation.AssistantTurnStatusFailed,
	),
	extraWhere:   "\n\t\t\tAND o.origin <> " + quoted(conversation.AssistantTurnOriginCompaction),
	agentColumn:  "NULL::varchar",
	decisionRun:  "o.run_id",
	taskFallback: quoted(aiprovider.TaskAssistantChat),
	outcomeSQL: fmt.Sprintf(`CASE
				WHEN o.status = %s THEN '%s'
				WHEN o.status = %s THEN '%s'
				WHEN COALESCE((o.fingerprint->>'stoppedAtLimit')::boolean, false) THEN '%s'
				WHEN h.run_id IS NOT NULL THEN '%s'
				ELSE '%s'
			END`,
		quoted(conversation.AssistantTurnStatusFailed), aianalytics.OutcomeFailed,
		quoted(conversation.AssistantTurnStatusStopped), aianalytics.OutcomeStopped,
		aianalytics.OutcomeStopped,
		aianalytics.OutcomeHandedOff,
		aianalytics.OutcomeCompleted,
	),
	startedAtSQL:  "COALESCE(NULLIF(o.started_at, 0), o.created_at) * 1000",
	finishedAtSQL: "COALESCE(o.completed_at, o.updated_at)",
	modelFallback: "",
})

var (
	callUnitKeySQL = fmt.Sprintf(
		"CASE WHEN u.trace_id IS NOT NULL THEN '%s' || md5(u.trace_id || ':' || u.task || ':' || COALESCE(u.subject_id, '')) ELSE u.id END",
		callUnitPrefix,
	)

	standaloneCallSQL = fmt.Sprintf(`u.organization_id = ?org AND u.business_unit_id = ?bu
			AND u.owner_id IS NULL AND u.run_id IS NULL AND u.thread_id IS NULL
			AND u.surface NOT IN (%s)
			AND COALESCE(u.feature, '') <> %s`,
		standaloneSurfaces,
		quoted(aiusage.FeatureAgentEvaluation),
	)

	callUnitsSQL = fmt.Sprintf(`WITH touched AS (
		SELECT DISTINCT %[1]s AS unit_id
		FROM ai_usage_records AS u
		WHERE %[2]s
			AND u.created_at >= ?from AND u.created_at < ?to
	),
	calls AS (
		SELECT %[1]s AS unit_id, %[3]s
		FROM ai_usage_records AS u
		WHERE %[2]s
			AND u.created_at >= ?from - %[4]d AND u.created_at < ?to
			AND %[1]s IN (SELECT unit_id FROM touched)
		GROUP BY 1
	)
	INSERT INTO ai_work_units (%[5]s)
	SELECT c.unit_id, ?bu, ?org, '%[6]s', c.agent_definition_id, c.provider_id, c.model, '',
		c.task,
		CASE WHEN c.answered THEN '%[7]s' ELSE '%[8]s' END,
		NULL,
		c.first_start_ms,
		c.last_call_at,
		GREATEST(0, c.last_end_ms - c.first_start_ms),
		c.first_token_ms, c.model_calls, c.failed_calls, c.failovers, c.input_tokens,
		c.output_tokens, c.cache_read_tokens, c.cache_write_tokens, c.cost_usd,
		c.advisor_calls, c.advisor_prompt_tokens, c.advisor_output_tokens, c.advisor_cost_usd,
		0, 0, 0, ?stamp
	FROM calls AS c
	WHERE c.model IS NOT NULL
	%[9]s`,
		callUnitKeySQL,
		standaloneCallSQL,
		callAggregates(),
		lookbackSeconds,
		unitColumns,
		aianalytics.SourceCall,
		aianalytics.OutcomeCompleted,
		aianalytics.OutcomeFailed,
		unitConflict,
	)
)

var (
	clearUnitToolsSQL = fmt.Sprintf(`DELETE FROM ai_work_unit_tools AS t
	USING ai_work_units AS w
	WHERE t.organization_id = ?org AND t.business_unit_id = ?bu
		AND w.organization_id = t.organization_id AND w.business_unit_id = t.business_unit_id
		AND w.id = t.unit_id AND w.refresh_stamp = ?stamp
		AND w.source IN (%s)`,
		quoted(aianalytics.SourceAgentRun, aianalytics.SourceAssistantTurn),
	)

	insertUnitToolsSQL = fmt.Sprintf(`INSERT INTO ai_work_unit_tools
		(organization_id, business_unit_id, unit_id, tool_name, cause, finished_at,
		model_key, provider_id, agent_definition_id, task, calls)
	SELECT ?org, ?bu, s.owner_id, s.tool_name, c.cause, w.finished_at,
		w.model_key, COALESCE(w.provider_id, ''), COALESCE(w.agent_definition_id, ''), w.task, COUNT(*)
	FROM ai_work_units AS w
	JOIN agent_run_steps AS s
		ON s.organization_id = w.organization_id
		AND s.business_unit_id = w.business_unit_id
		AND s.owner_id = w.id
	CROSS JOIN LATERAL (SELECT %[1]s AS cause) AS c
	WHERE w.organization_id = ?org AND w.business_unit_id = ?bu
		AND w.refresh_stamp = ?stamp
		AND w.source IN (%[2]s)
		AND s.kind = '%[3]s' AND s.status <> '%[4]s' AND s.tool_name <> ''
	GROUP BY s.owner_id, s.tool_name, c.cause, w.finished_at, w.model_key, w.provider_id,
		w.agent_definition_id, w.task`,
		causeSQL,
		quoted(aianalytics.SourceAgentRun, aianalytics.SourceAssistantTurn),
		agentports.RunStepTool,
		agentports.RunStepStarted,
	)
)

var (
	clearProviderHoursSQL = `DELETE FROM ai_provider_hourly_stats
	WHERE organization_id = ?org AND business_unit_id = ?bu
		AND hour >= ?from_hour AND hour < ?to_hour`

	insertProviderHoursSQL = fmt.Sprintf(`WITH calls AS (
		SELECT (u.created_at / 3600) * 3600 AS hour,
			COALESCE(u.provider_id, '') AS provider_id,
			u.model,
			u.task,
			COALESCE(u.agent_definition_id, '') AS agent_definition_id,
			%[1]s AS error_class,
			u.succeeded,
			u.first_token_ms,
			u.trace_id
		FROM ai_usage_records AS u
		WHERE u.organization_id = ?org AND u.business_unit_id = ?bu
			AND u.created_at >= ?from_hour AND u.created_at < ?to_hour
			AND u.surface NOT IN (%[2]s)
	),
	rescued AS (
		SELECT trace_id, ARRAY_AGG(DISTINCT provider_id) AS providers
		FROM calls
		WHERE succeeded AND trace_id IS NOT NULL
		GROUP BY trace_id
	),
	grouped AS (
		SELECT c.hour, c.provider_id, c.model, c.task, c.agent_definition_id, c.error_class,
			COUNT(*) AS calls,
			COUNT(*) FILTER (
				WHERE NOT c.succeeded AND r.providers IS NOT NULL
					AND NOT (r.providers <@ ARRAY[c.provider_id]::varchar[])
			) AS fell_to_next
		FROM calls AS c
		LEFT JOIN rescued AS r ON r.trace_id = c.trace_id
		GROUP BY c.hour, c.provider_id, c.model, c.task, c.agent_definition_id, c.error_class
	),
	hist AS (
		SELECT c.hour, c.provider_id, c.model, c.task, c.agent_definition_id,
			%[3]s AS bucket,
			COUNT(*)::int AS n
		FROM calls AS c
		WHERE c.succeeded AND c.first_token_ms IS NOT NULL
		GROUP BY 1, 2, 3, 4, 5, 6
	),
	arrays AS (
		SELECT k.hour, k.provider_id, k.model, k.task, k.agent_definition_id,
			ARRAY_AGG(COALESCE(h.n, 0) ORDER BY b.i) AS first_token_hist
		FROM (SELECT DISTINCT hour, provider_id, model, task, agent_definition_id FROM hist) AS k
		CROSS JOIN generate_series(0, %[4]d) AS b(i)
		LEFT JOIN hist AS h
			ON h.hour = k.hour AND h.provider_id = k.provider_id AND h.model = k.model
			AND h.task = k.task AND h.agent_definition_id = k.agent_definition_id
			AND h.bucket = b.i
		GROUP BY k.hour, k.provider_id, k.model, k.task, k.agent_definition_id
	)
	INSERT INTO ai_provider_hourly_stats
		(organization_id, business_unit_id, hour, provider_id, model, model_key, task,
		agent_definition_id, error_class, calls, fell_to_next, first_token_hist)
	SELECT ?org, ?bu, g.hour, g.provider_id, g.model, '', g.task, g.agent_definition_id,
		g.error_class, g.calls, g.fell_to_next,
		CASE WHEN g.error_class = '' THEN COALESCE(a.first_token_hist, '{}') ELSE '{}' END
	FROM grouped AS g
	LEFT JOIN arrays AS a
		ON a.hour = g.hour AND a.provider_id = g.provider_id AND a.model = g.model
		AND a.task = g.task AND a.agent_definition_id = g.agent_definition_id`,
		providerErrorSQL,
		nonCompletionSurfaces,
		bucketSQL,
		aianalytics.HistogramBuckets-1,
	)
)
