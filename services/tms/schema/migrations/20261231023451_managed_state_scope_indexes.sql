--keel:notx
-- The plan history is read and pruned per scope, newest first.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_managed_feature_changes_scope_changed_at"
    ON "managed_feature_changes"("scope_key", "changed_at" DESC, "id" DESC);
--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_managed_feature_changes_changed_at";
--keel:split
-- Whether any active organization on a shared server has a feature: the question schedules and
-- other work that belongs to no tenant ask.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_managed_feature_states_active_features"
    ON "managed_feature_states" USING gin("features" jsonb_path_ops)
    WHERE "lifecycle" = 'active';
--keel:split
-- The hourly reminder finds the active scopes whose plan ends within three days.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_managed_feature_states_plan_ends_at"
    ON "managed_feature_states"("plan_ends_at")
    WHERE "plan_ends_at" IS NOT NULL AND "lifecycle" = 'active';
