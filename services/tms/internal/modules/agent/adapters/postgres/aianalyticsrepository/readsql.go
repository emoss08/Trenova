package aianalyticsrepository

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/settingversion"
)

const (
	unitScopeSQL = `w.organization_id = ?org AND w.business_unit_id = ?bu
		AND (?agent = '' OR w.agent_definition_id = ?agent)
		AND (?task = '' OR w.task = ?task)`
	unitWindowSQL = unitScopeSQL + `
		AND w.finished_at >= ?since`
	hourScopeSQL = `h.organization_id = ?org AND h.business_unit_id = ?bu
		AND h.hour >= ?since_hour
		AND (?agent = '' OR h.agent_definition_id = ?agent)
		AND (?task = '' OR h.task = ?task)`

	modelKeySQL    = "w.model_key"
	providerKeySQL = "COALESCE(w.provider_id, '')"

	hourModelKeySQL    = "h.model_key"
	hourProviderKeySQL = "h.provider_id"

	deploymentColumnsSQL     = modelKeySQL + " AS model_key, " + providerKeySQL + " AS provider_id"
	hourDeploymentColumnsSQL = hourModelKeySQL + " AS model_key, " + hourProviderKeySQL + " AS provider_id"
)

func keySQL(compareBy aianalytics.CompareBy) string {
	if compareBy == aianalytics.CompareByProvider {
		return providerKeySQL
	}

	return modelKeySQL
}

func groupColumnsSQL(grouping aianalytics.Grouping) string {
	switch grouping {
	case aianalytics.GroupingProvider:
		return "''::varchar AS model_key, " + providerKeySQL + " AS provider_id"
	case aianalytics.GroupingDeployment:
		return deploymentColumnsSQL
	default:
		return modelKeySQL + " AS model_key, ''::varchar AS provider_id"
	}
}

var outcomeCountsSQL = fmt.Sprintf(`COUNT(*) AS runs,
		COUNT(*) FILTER (WHERE w.outcome = '%[1]s') AS completed,
		COUNT(*) FILTER (WHERE w.outcome = '%[2]s') AS handed_off,
		COUNT(*) FILTER (WHERE w.outcome = '%[3]s') AS stopped,
		COUNT(*) FILTER (WHERE w.outcome = '%[4]s') AS failed,
		COALESCE(SUM(w.model_calls), 0) AS model_calls,
		COALESCE(SUM(w.failed_calls), 0) AS failed_calls,
		COALESCE(SUM(w.tool_calls), 0) AS tool_calls,
		COALESCE(SUM(w.tool_failures_model), 0) AS tool_failures_model,
		COALESCE(SUM(w.tool_failures_tool), 0) AS tool_failures_tool,
		COALESCE(SUM(w.cost_usd), 0)::text AS cost_usd,
		COUNT(w.cost_usd) AS priced_runs,
		COALESCE(MAX(w.finished_at), 0) AS last_ran_at`,
	aianalytics.OutcomeCompleted,
	aianalytics.OutcomeHandedOff,
	aianalytics.OutcomeStopped,
	aianalytics.OutcomeFailed,
)

var totalsSQL = fmt.Sprintf(`SELECT %[1]s,
		COUNT(*) FILTER (WHERE w.outcome = '%[2]s') AS completion_count,
		percentile_cont(ARRAY[0.25, 0.5, 0.75, 0.95]) WITHIN GROUP (ORDER BY w.duration_ms)
			FILTER (WHERE w.outcome = '%[2]s') AS completion
	FROM ai_work_units AS w
	WHERE %[3]s`,
	outcomeCountsSQL,
	aianalytics.OutcomeCompleted,
	unitWindowSQL,
)

var deploymentsSQL = fmt.Sprintf(`SELECT %[1]s, %[2]s
	FROM ai_work_units AS w
	WHERE %[3]s
	GROUP BY 1, 2`,
	deploymentColumnsSQL, outcomeCountsSQL, unitWindowSQL)

func outcomeDurationsSQL(grouping aianalytics.Grouping) string {
	return fmt.Sprintf(`SELECT %[1]s, w.outcome,
		COUNT(*) AS count,
		percentile_cont(ARRAY[0.25, 0.5, 0.75, 0.95]) WITHIN GROUP (ORDER BY w.duration_ms) AS percentiles
	FROM ai_work_units AS w
	WHERE %[2]s
	GROUP BY 1, 2, 3`,
		groupColumnsSQL(grouping), unitWindowSQL)
}

func offsetSQL(spans []aianalyticsports.OffsetSpan) string {
	if len(spans) == 0 {
		return "0"
	}
	if len(spans) == 1 {
		return strconv.FormatInt(spans[0].Offset, 10)
	}
	var builder strings.Builder
	builder.WriteString("CASE")
	for idx := len(spans) - 1; idx > 0; idx-- {
		fmt.Fprintf(
			&builder,
			" WHEN w.finished_at >= %d THEN %d",
			spans[idx].From,
			spans[idx].Offset,
		)
	}
	fmt.Fprintf(&builder, " ELSE %d END", spans[0].Offset)

	return builder.String()
}

func trendSQL(grouping aianalytics.Grouping, spans []aianalyticsports.OffsetSpan) string {
	return fmt.Sprintf(`SELECT %[1]s,
		(w.finished_at + o.off) - (w.finished_at + o.off) %% ?bucket - o.off AS bucket_start,
		COUNT(*) AS runs,
		COUNT(*) FILTER (WHERE w.outcome = '%[2]s') AS completed,
		COALESCE(SUM(w.tool_calls), 0) AS tool_calls,
		COALESCE(SUM(w.tool_failures_model + w.tool_failures_tool), 0) AS tool_failures,
		percentile_cont(0.5) WITHIN GROUP (ORDER BY w.duration_ms)
			FILTER (WHERE w.outcome = '%[2]s') AS median_run_ms
	FROM ai_work_units AS w
	CROSS JOIN LATERAL (SELECT (%[4]s)::bigint AS off) AS o
	WHERE %[3]s
	GROUP BY 1, 2, 3`,
		groupColumnsSQL(grouping), aianalytics.OutcomeCompleted, unitWindowSQL, offsetSQL(spans))
}

const toolScopeSQL = `t.organization_id = ?org AND t.business_unit_id = ?bu AND t.finished_at >= ?since
		AND (?agent = '' OR t.agent_definition_id = ?agent)
		AND (?task = '' OR t.task = ?task)`

var toolCellsSQL = `SELECT t.model_key, t.provider_id, t.tool_name, t.cause,
		SUM(t.calls) AS calls
	FROM ai_work_unit_tools AS t
	WHERE ` + toolScopeSQL + `
	GROUP BY 1, 2, 3, 4`

func toolKeySQL(compareBy aianalytics.CompareBy) string {
	if compareBy == aianalytics.CompareByProvider {
		return "t.provider_id"
	}

	return "t.model_key"
}

var taskCellsSQL = fmt.Sprintf(`SELECT %[1]s, w.task,
		COUNT(*) AS runs,
		COUNT(*) FILTER (WHERE w.outcome = '%[2]s') AS completed
	FROM ai_work_units AS w
	WHERE %[3]s
	GROUP BY 1, 2, 3`,
	deploymentColumnsSQL, aianalytics.OutcomeCompleted, unitWindowSQL)

var decisionsSQL = fmt.Sprintf(`SELECT %[1]s,
		COUNT(*) FILTER (WHERE d.decision = '%[2]s') AS accepted,
		COUNT(*) FILTER (WHERE d.decision = '%[3]s') AS modified,
		COUNT(*) FILTER (WHERE d.decision = '%[4]s') AS rejected
	FROM agent_decisions AS d
	JOIN agent_proposals AS p
		ON p.organization_id = d.organization_id
		AND p.business_unit_id = d.business_unit_id
		AND p.id = d.proposal_id
	JOIN ai_work_units AS w
		ON w.organization_id = p.organization_id
		AND w.business_unit_id = p.business_unit_id
		AND w.decision_run_id = p.run_id
	WHERE d.organization_id = ?org AND d.business_unit_id = ?bu
		AND d.created_at >= ?since
		AND d.proposal_id IS NOT NULL
		AND d.undone_at IS NULL
		AND %[5]s
	GROUP BY 1, 2`,
	deploymentColumnsSQL,
	agent.DecisionAccepted,
	agent.DecisionModified,
	agent.DecisionRejected,
	unitScopeSQL,
)

var ratingsSQL = fmt.Sprintf(`SELECT %[1]s,
		COUNT(*) AS rated,
		COUNT(*) FILTER (WHERE f.rating > 0) AS helpful
	FROM ai_feedback AS f
	JOIN ai_work_units AS w
		ON w.organization_id = f.organization_id
		AND w.business_unit_id = f.business_unit_id
		AND w.id = COALESCE(f.turn_id, f.run_id)
	WHERE f.organization_id = ?org AND f.business_unit_id = ?bu
		AND f.created_at >= ?since
		AND %[2]s
	GROUP BY 1, 2`,
	deploymentColumnsSQL, unitScopeSQL)

var providerCallsSQL = `SELECT ` + hourDeploymentColumnsSQL + `, h.error_class,
		SUM(h.calls) AS calls,
		SUM(h.fell_to_next) AS fell_to_next,
		MAX(h.hour) AS last_hour
	FROM ai_provider_hourly_stats AS h
	WHERE ` + hourScopeSQL + `
	GROUP BY 1, 2, 3`

var firstTokenSQL = fmt.Sprintf(`SELECT %[1]s,
		(b.idx - 1)::int AS bucket,
		SUM(b.v)::bigint AS count
	FROM ai_provider_hourly_stats AS h
	CROSS JOIN LATERAL unnest(h.first_token_hist) WITH ORDINALITY AS b(v, idx)
	WHERE %[2]s AND h.error_class = ''
	GROUP BY 1, 2, 3
	HAVING SUM(b.v) > 0`,
	hourDeploymentColumnsSQL, hourScopeSQL)

var providerHoursSQL = fmt.Sprintf(`SELECT h.provider_id, h.hour,
		SUM(h.calls) AS calls,
		COALESCE(SUM(h.calls) FILTER (WHERE h.error_class IN (%[1]s)), 0) AS failed,
		COALESCE(SUM(h.calls) FILTER (WHERE h.error_class = '%[2]s'), 0) AS auth_failed
	FROM ai_provider_hourly_stats AS h
	WHERE %[3]s AND h.provider_id <> ''
	GROUP BY 1, 2
	ORDER BY 1, 2`,
	quoted(aianalytics.AllProviderErrors()...), aianalytics.ProviderErrorAccessDenied, hourScopeSQL)

var providerVersionsSQL = fmt.Sprintf(`SELECT v.subject_id AS provider_id,
		v.version,
		COALESCE(v.snapshot->>'name', '') AS name,
		COALESCE(v.snapshot->>'model', '') AS model,
		COALESCE((v.snapshot->>'enabled')::boolean, false) AS enabled,
		v.created_at
	FROM ai_setting_versions AS v
	WHERE v.organization_id = ?org AND v.business_unit_id = ?bu
		AND v.kind = '%[1]s'
		AND v.subject_id IN (
			SELECT x.subject_id FROM ai_setting_versions AS x
			WHERE x.organization_id = ?org AND x.business_unit_id = ?bu
				AND x.kind = '%[1]s' AND x.created_at >= ?since
		)
	ORDER BY v.subject_id, v.version`,
	settingversion.KindAIProvider)

func toolCausesSQL(compareBy aianalytics.CompareBy) string {
	return fmt.Sprintf(`SELECT t.cause, SUM(t.calls) AS calls
	FROM ai_work_unit_tools AS t
	WHERE %[1]s
		AND t.tool_name = ?tool_name
		AND %[2]s = ?series_key
	GROUP BY 1`,
		toolScopeSQL, toolKeySQL(compareBy))
}

func toolExamplesSQL(compareBy aianalytics.CompareBy) string {
	return fmt.Sprintf(`SELECT w.id AS unit_id, w.source, s.tool_name,
		LEFT(COALESCE(NULLIF(s.outcome->>'content', ''), s.outcome->>'reason', ''), 300) AS message,
		COALESCE(w.agent_definition_id, '') AS agent_definition_id,
		COALESCE(s.settled_at_ms / 1000, s.updated_at) AS at
	FROM ai_work_units AS w
	JOIN agent_run_steps AS s
		ON s.organization_id = w.organization_id
		AND s.business_unit_id = w.business_unit_id
		AND s.owner_id = w.id
	WHERE %[1]s
		AND %[2]s = ?series_key
		AND s.kind = '%[3]s' AND s.status = '%[4]s' AND s.tool_name = ?tool_name
	ORDER BY s.updated_at DESC, s.id DESC
	LIMIT ?limit`,
		unitWindowSQL, keySQL(compareBy), agentports.RunStepTool, agentports.RunStepFailed)
}

const paletteSQL = `SELECT DISTINCT h.model_key, h.provider_id
	FROM ai_provider_hourly_stats AS h
	WHERE h.organization_id = ?org AND h.business_unit_id = ?bu AND h.hour >= ?since_hour`
