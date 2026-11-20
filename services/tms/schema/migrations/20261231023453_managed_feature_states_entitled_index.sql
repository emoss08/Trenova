--keel:notx
-- The organizations a shared server's plans entitle to background work: an active plan the
-- control plane has sent. Fan-outs and sweeps across organizations keep their rows to these
-- (entitlement filter); the add-on case also reads idx_managed_feature_states_active_features.
-- It also serves the question whether any organization has an active plan at all.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_managed_feature_states_entitled"
    ON "managed_feature_states"("scope_key")
    WHERE "lifecycle" = 'active' AND "received_at" > 0;
