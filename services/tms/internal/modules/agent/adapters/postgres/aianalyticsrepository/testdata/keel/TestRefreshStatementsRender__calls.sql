INSERT INTO ai_work_units AS awu (id, source, agent_definition_id, provider_id, model, model_key, task, outcome, started_at_ms, finished_at, duration_ms, first_token_ms, model_calls, failed_calls, failovers, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, cost_usd, advisor_calls, advisor_prompt_tokens, advisor_output_tokens, advisor_cost_usd, tool_calls, tool_failures_model, tool_failures_tool, refresh_stamp, business_unit_id, organization_id)
SELECT s.c1, s.c2, nullif(s.c3, ''), nullif(s.c4, ''), s.c5, s.c6, s.c7, s.c8, s.c9, s.c10, s.c11, s.c12, s.c13, s.c14, s.c15, s.c16, s.c17, s.c18, s.c19, s.c20, s.c21, s.c22, s.c23, s.c24, s.c25, s.c26, s.c27, s.c28, $1, $2
FROM (
  WITH touched (unit_id) AS (
    SELECT DISTINCT CASE WHEN aur.trace_id IS NOT NULL THEN ($12 || md5(((((aur.trace_id || $13) || aur.task) || $14) || coalesce(aur.subject_id, $15)))) ELSE aur.id END
    FROM ai_usage_records aur
    WHERE aur.business_unit_id = $1 AND aur.organization_id = $2 AND aur.owner_id IS NULL AND aur.run_id IS NULL AND aur.thread_id IS NULL AND aur.surface NOT IN ('Evaluation', 'Indexing', 'Retrieval') AND coalesce(aur.feature, $8) <> $9 AND aur.created_at >= $10 AND aur.created_at < $11
  ), calls (unit_id, model_calls, failed_calls, failovers, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, cost_usd, advisor_calls, advisor_prompt_tokens, advisor_output_tokens, advisor_cost_usd, first_start_ms, last_end_ms, first_token_ms, model, provider_id, task, agent_definition_id, answered, last_call_at) AS (
    SELECT CASE WHEN aur2.trace_id IS NOT NULL THEN ($12 || md5(((((aur2.trace_id || $13) || aur2.task) || $14) || coalesce(aur2.subject_id, $15)))) ELSE aur2.id END, count(*), count(*) FILTER (WHERE aur2.succeeded = $18), count(*) FILTER (WHERE aur2.failover IS TRUE), coalesce(sum(CASE WHEN aur2.provider_kind = 'AnthropicMessages' THEN aur2.input_tokens ELSE greatest(((aur2.input_tokens - aur2.cache_read_tokens) - aur2.cache_write_tokens), 0) END), $19), coalesce(sum(aur2.output_tokens), $20), coalesce(sum(aur2.cache_read_tokens), $21), coalesce(sum(aur2.cache_write_tokens), $22), sum(aur2.cost_usd), coalesce(sum(aur2.advisor_calls), $23), coalesce(sum(((aur2.advisor_input_tokens + aur2.advisor_cache_read_tokens) + aur2.advisor_cache_write_tokens)), $24), coalesce(sum(aur2.advisor_output_tokens), $25), sum(aur2.advisor_cost_usd), min((coalesce(aur2.recorded_at_ms, (aur2.created_at * $26::bigint)) - aur2.latency_ms)), max(coalesce(aur2.recorded_at_ms, (aur2.created_at * $26::bigint))), (array_agg(aur2.first_token_ms ORDER BY (coalesce(aur2.recorded_at_ms, (aur2.created_at * $26::bigint)) - aur2.latency_ms) ASC) FILTER (WHERE aur2.first_token_ms IS NOT NULL))[1], (array_agg(aur2.model ORDER BY aur2.succeeded DESC, coalesce(aur2.recorded_at_ms, (aur2.created_at * $26::bigint)) DESC))[1], (array_agg(aur2.provider_id ORDER BY aur2.succeeded DESC, coalesce(aur2.recorded_at_ms, (aur2.created_at * $26::bigint)) DESC))[1], (array_agg(aur2.task ORDER BY aur2.succeeded DESC, coalesce(aur2.recorded_at_ms, (aur2.created_at * $26::bigint)) DESC))[1], (array_agg(aur2.agent_definition_id ORDER BY coalesce(aur2.recorded_at_ms, (aur2.created_at * $26::bigint)) ASC) FILTER (WHERE aur2.agent_definition_id IS NOT NULL))[1], bool_or(aur2.succeeded), max(aur2.created_at)
    FROM ai_usage_records aur2
    WHERE aur2.business_unit_id = $1 AND aur2.organization_id = $2 AND aur2.owner_id IS NULL AND aur2.run_id IS NULL AND aur2.thread_id IS NULL AND aur2.surface NOT IN ('Evaluation', 'Indexing', 'Retrieval') AND coalesce(aur2.feature, $8) <> $16 AND aur2.created_at >= $17 AND aur2.created_at < $11 AND EXISTS (
      SELECT 1
      FROM touched t
      WHERE t.unit_id = CASE WHEN aur2.trace_id IS NOT NULL THEN ($12 || md5(((((aur2.trace_id || $13) || aur2.task) || $14) || coalesce(aur2.subject_id, $15)))) ELSE aur2.id END
    )
    GROUP BY 1
  )
  SELECT c.unit_id, $3::text, c.agent_definition_id, c.provider_id, c.model, $4::text, c.task, CASE WHEN c.answered IS TRUE THEN $5::text ELSE $6::text END, c.first_start_ms, c.last_call_at, greatest(0, (c.last_end_ms - c.first_start_ms)), c.first_token_ms, c.model_calls, c.failed_calls, c.failovers, c.input_tokens, c.output_tokens, c.cache_read_tokens, c.cache_write_tokens, c.cost_usd, c.advisor_calls, c.advisor_prompt_tokens, c.advisor_output_tokens, c.advisor_cost_usd, 0, 0, 0, $7::bigint
  FROM calls c
  WHERE c.model IS NOT NULL
) AS s(c1, c2, c3, c4, c5, c6, c7, c8, c9, c10, c11, c12, c13, c14, c15, c16, c17, c18, c19, c20, c21, c22, c23, c24, c25, c26, c27, c28)
ON CONFLICT (id, business_unit_id, organization_id) DO UPDATE SET source = EXCLUDED.source, agent_definition_id = EXCLUDED.agent_definition_id, provider_id = EXCLUDED.provider_id, model = EXCLUDED.model, task = EXCLUDED.task, outcome = EXCLUDED.outcome, decision_run_id = EXCLUDED.decision_run_id, started_at_ms = EXCLUDED.started_at_ms, finished_at = EXCLUDED.finished_at, duration_ms = EXCLUDED.duration_ms, first_token_ms = EXCLUDED.first_token_ms, model_calls = EXCLUDED.model_calls, failed_calls = EXCLUDED.failed_calls, failovers = EXCLUDED.failovers, input_tokens = EXCLUDED.input_tokens, output_tokens = EXCLUDED.output_tokens, cache_read_tokens = EXCLUDED.cache_read_tokens, cache_write_tokens = EXCLUDED.cache_write_tokens, cost_usd = EXCLUDED.cost_usd, advisor_calls = EXCLUDED.advisor_calls, advisor_prompt_tokens = EXCLUDED.advisor_prompt_tokens, advisor_output_tokens = EXCLUDED.advisor_output_tokens, advisor_cost_usd = EXCLUDED.advisor_cost_usd, tool_calls = EXCLUDED.tool_calls, tool_failures_model = EXCLUDED.tool_failures_model, tool_failures_tool = EXCLUDED.tool_failures_tool, refresh_stamp = EXCLUDED.refresh_stamp, model_key = CASE WHEN awu.model = EXCLUDED.model THEN awu.model_key ELSE $27 END
WHERE (ROW(awu.source, awu.agent_definition_id, awu.provider_id, awu.model, awu.task, awu.outcome, awu.decision_run_id, awu.started_at_ms, awu.finished_at, awu.duration_ms, awu.first_token_ms, awu.model_calls, awu.failed_calls, awu.failovers, awu.input_tokens, awu.output_tokens, awu.cache_read_tokens, awu.cache_write_tokens, awu.cost_usd, awu.advisor_calls, awu.advisor_prompt_tokens, awu.advisor_output_tokens, awu.advisor_cost_usd, awu.tool_calls, awu.tool_failures_model, awu.tool_failures_tool) IS DISTINCT FROM ROW(EXCLUDED.source, EXCLUDED.agent_definition_id, EXCLUDED.provider_id, EXCLUDED.model, EXCLUDED.task, EXCLUDED.outcome, EXCLUDED.decision_run_id, EXCLUDED.started_at_ms, EXCLUDED.finished_at, EXCLUDED.duration_ms, EXCLUDED.first_token_ms, EXCLUDED.model_calls, EXCLUDED.failed_calls, EXCLUDED.failovers, EXCLUDED.input_tokens, EXCLUDED.output_tokens, EXCLUDED.cache_read_tokens, EXCLUDED.cache_write_tokens, EXCLUDED.cost_usd, EXCLUDED.advisor_calls, EXCLUDED.advisor_prompt_tokens, EXCLUDED.advisor_output_tokens, EXCLUDED.advisor_cost_usd, EXCLUDED.tool_calls, EXCLUDED.tool_failures_model, EXCLUDED.tool_failures_tool))
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 'Call'
-- $4 = ''
-- $5 = 'Completed'
-- $6 = 'Failed'
-- $7 = 7
-- $8 = ''
-- $9 = 'AgentEvaluation'
-- $10 = 1790000000
-- $11 = 1790003600
-- $12 = 'call_'
-- $13 = ':'
-- $14 = ':'
-- $15 = ''
-- $16 = 'AgentEvaluation'
-- $17 = 1789996400
-- $18 = false
-- $19 = 0
-- $20 = 0
-- $21 = 0
-- $22 = 0
-- $23 = 0
-- $24 = 0
-- $25 = 0
-- $26 = 1000
-- $27 = ''
