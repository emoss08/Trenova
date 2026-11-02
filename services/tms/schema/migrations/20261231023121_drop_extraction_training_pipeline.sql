-- Nova reads documents on its own pinned model, so there is no second extraction model to
-- evaluate, shadow, roll out or train. The evaluation cases and runs, shadow settings and
-- results, rollouts and their assignments, training exports and retraining cycles go, with the
-- consent that gated sharing corrections for training. Corrections stay: production accuracy
-- and drift are read from them.
DROP TABLE IF EXISTS "extraction_eval_results", "extraction_eval_runs", "extraction_eval_case_pages", "extraction_eval_cases", "extraction_shadow_results", "extraction_shadow_settings", "extraction_rollout_assignments", "extraction_rollout_exclusions", "extraction_rollouts", "ai_training_export_records", "ai_training_exports", "ai_retraining_cycles";

ALTER TABLE "agent_controls"
    DROP COLUMN IF EXISTS "ai_training_consent",
    DROP COLUMN IF EXISTS "ai_training_consent_changed_at",
    DROP COLUMN IF EXISTS "ai_training_consent_changed_by_id";

DELETE FROM trenova_rls.global_tables
WHERE "table_name" IN ('ai_retraining_cycles', 'ai_training_exports', 'ai_training_export_records');
