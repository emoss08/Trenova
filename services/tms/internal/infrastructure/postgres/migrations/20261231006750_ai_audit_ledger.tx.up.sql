-- The AI audit trail: what agents did, for whom, and who decided, written
-- once by the audit projector from rows the runtime already keeps.
--
-- Each tenant's rows form a hash chain in seq order. The projector is the only
-- writer: it takes a row lock on the tenant's chain head, assigns the next seq
-- and links each row to the one before it, signed with a key kept outside the
-- database. A trigger refuses every update, and every delete that is not the
-- retention sweep.
CREATE TABLE IF NOT EXISTS "ai_audit_events"(
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "seq" BIGINT NOT NULL,
    "source_key" VARCHAR(250) NOT NULL,
    "occurred_at" BIGINT NOT NULL,
    "recorded_at" BIGINT NOT NULL,
    "kind" VARCHAR(30) NOT NULL,
    "outcome" VARCHAR(30) NOT NULL,
    "principal_type" VARCHAR(20) NOT NULL,
    "principal_id" VARCHAR(100),
    "on_behalf_of_user_id" VARCHAR(100),
    "on_behalf_of_user_name" VARCHAR(255),
    "decided_by_user_id" VARCHAR(100),
    "decided_by_user_name" VARCHAR(255),
    "agent_definition_id" VARCHAR(100),
    "agent_definition_version" BIGINT,
    "agent_name" VARCHAR(100),
    "owner_kind" VARCHAR(20),
    "owner_id" VARCHAR(100),
    "run_id" VARCHAR(100),
    "turn_id" VARCHAR(100),
    "thread_id" VARCHAR(100),
    "proposal_id" VARCHAR(100),
    "plan_id" VARCHAR(100),
    "decision_id" VARCHAR(100),
    "step_key" VARCHAR(120),
    "call_id" VARCHAR(200),
    "delegate_call_id" VARCHAR(200),
    "parent_owner_id" VARCHAR(100),
    "trace_id" VARCHAR(32),
    "span_id" VARCHAR(16),
    "provider_id" VARCHAR(100),
    "provider_kind" VARCHAR(50),
    "model" VARCHAR(200),
    "attempt" INTEGER,
    "failover" BOOLEAN NOT NULL DEFAULT FALSE,
    "input_tokens" INTEGER NOT NULL DEFAULT 0,
    "output_tokens" INTEGER NOT NULL DEFAULT 0,
    "reasoning_tokens" INTEGER NOT NULL DEFAULT 0,
    "cache_read_tokens" INTEGER NOT NULL DEFAULT 0,
    "cache_write_tokens" INTEGER NOT NULL DEFAULT 0,
    "cost_usd" NUMERIC(14, 6),
    "latency_ms" BIGINT,
    "tool_name" VARCHAR(200),
    "tool_effect" VARCHAR(20),
    "egress_class" VARCHAR(30),
    "tier" VARCHAR(30),
    "tier_source" VARCHAR(30),
    "held_by" TEXT[] NOT NULL DEFAULT '{}',
    "reason" TEXT,
    "arguments" JSONB,
    "argument_sensitivity" JSONB,
    "redacted_paths" TEXT[] NOT NULL DEFAULT '{}',
    "arguments_truncated" BOOLEAN NOT NULL DEFAULT FALSE,
    "result_summary" VARCHAR(500),
    "entity_type" VARCHAR(100),
    "entity_id" VARCHAR(100),
    "version_before" BIGINT,
    "version_after" BIGINT,
    "window_start" BIGINT,
    "window_end" BIGINT,
    "tainted" BOOLEAN NOT NULL DEFAULT FALSE,
    "taint" JSONB,
    "external_content" BOOLEAN NOT NULL DEFAULT FALSE,
    "simulated" BOOLEAN NOT NULL DEFAULT FALSE,
    "purpose" VARCHAR(20) NOT NULL DEFAULT 'Live',
    "reconstructed" BOOLEAN NOT NULL DEFAULT FALSE,
    "prev_hash" VARCHAR(64) NOT NULL,
    "hash" VARCHAR(64) NOT NULL,
    "hash_key_id" VARCHAR(40),
    "hash_version" SMALLINT NOT NULL,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_ai_audit_events" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "ck_ai_audit_events_kind" CHECK ("kind" IN ('RunStarted', 'RunEnded', 'ModelCall', 'ToolCall', 'ToolRefused', 'ProposalFiled', 'ProposalDecided', 'ProposalExecuted', 'ProposalExecutionFailed', 'ProposalSimulated', 'ProposalExpired', 'DelegationStarted', 'DelegationEnded')),
    CONSTRAINT "ck_ai_audit_events_outcome" CHECK ("outcome" IN ('Started', 'Completed', 'Failed', 'Refused', 'Stopped', 'Succeeded', 'Ran', 'Proposed', 'Simulated', 'Denied', 'Unknown', 'Filed', 'Accepted', 'Modified', 'Rejected', 'Expired', 'Exhausted', 'Declined')),
    CONSTRAINT "ck_ai_audit_events_principal_type" CHECK ("principal_type" IN ('User', 'Agent', 'System')),
    CONSTRAINT "ck_ai_audit_events_purpose" CHECK ("purpose" IN ('Live', 'Evaluation')),
    CONSTRAINT "ck_ai_audit_events_owner_kind" CHECK ("owner_kind" IS NULL OR "owner_kind" IN ('AgentRun', 'AssistantTurn')),
    CONSTRAINT "ck_ai_audit_events_hash_version" CHECK ("hash_version" IN (1, 2)),
    CONSTRAINT "ck_ai_audit_events_seq" CHECK ("seq" >= 1),
    CONSTRAINT "ck_ai_audit_events_signed_key" CHECK (("hash_version" = 1) = ("hash_key_id" IS NOT NULL)),
    CONSTRAINT "fk_ai_audit_events_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_audit_events_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

COMMENT ON TABLE "ai_audit_events" IS 'The AI audit trail: append-only, hash-chained per tenant, written only by the audit projector';

COMMENT ON COLUMN "ai_audit_events"."source_key" IS 'The source row and moment the event was projected from, unique per tenant so a re-scan inserts nothing twice';

COMMENT ON COLUMN "ai_audit_events"."arguments" IS 'The call arguments, redacted when projected: confidential fields replaced, sensitive patterns masked, bounded to 16 KiB';

COMMENT ON COLUMN "ai_audit_events"."argument_sensitivity" IS 'The resource the tool acts on and each argument path''s sensitivity, so readers are shown only what their role reaches';

COMMENT ON COLUMN "ai_audit_events"."reconstructed" IS 'Provenance was worked out from source rows written before their link columns existed';

COMMENT ON COLUMN "ai_audit_events"."hash" IS 'HMAC-SHA256 (hash_version 1) or SHA-256 (hash_version 2, no key configured) over the canonical row chained with prev_hash';

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_ai_audit_events_source_key" ON "ai_audit_events"("organization_id", "business_unit_id", "source_key");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_ai_audit_events_seq" ON "ai_audit_events"("organization_id", "business_unit_id", "seq");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_occurred" ON "ai_audit_events"("organization_id", "business_unit_id", "occurred_at" DESC, "id" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_agent" ON "ai_audit_events"("organization_id", "business_unit_id", "agent_definition_id", "occurred_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_tool" ON "ai_audit_events"("organization_id", "business_unit_id", "tool_name", "occurred_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_entity" ON "ai_audit_events"("organization_id", "business_unit_id", "entity_type", "entity_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_on_behalf_of" ON "ai_audit_events"("organization_id", "business_unit_id", "on_behalf_of_user_id", "occurred_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_decided_by" ON "ai_audit_events"("organization_id", "business_unit_id", "decided_by_user_id") WHERE "decided_by_user_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_owner" ON "ai_audit_events"("organization_id", "business_unit_id", "owner_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_trace" ON "ai_audit_events"("organization_id", "business_unit_id", "trace_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_proposal" ON "ai_audit_events"("organization_id", "business_unit_id", "proposal_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_events_created" ON "ai_audit_events"("created_at");

--bun:split

-- Rows are never changed. The retention sweep deletes only after setting
-- trenova.ai_audit_prune for its own transaction, and a tenant's own removal
-- cascades from inside another trigger; every other delete is refused.
CREATE OR REPLACE FUNCTION "ai_audit_events_append_only"()
    RETURNS TRIGGER
    AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        RAISE EXCEPTION 'ai_audit_events rows cannot be changed'
            USING ERRCODE = 'integrity_constraint_violation';
    END IF;
    IF TG_OP = 'TRUNCATE' THEN
        RAISE EXCEPTION 'ai_audit_events cannot be truncated'
            USING ERRCODE = 'integrity_constraint_violation';
    END IF;
    IF COALESCE(current_setting('trenova.ai_audit_prune', TRUE), '') <> 'on' AND pg_trigger_depth() <= 1 THEN
        RAISE EXCEPTION 'ai_audit_events rows are removed only by the retention sweep'
            USING ERRCODE = 'integrity_constraint_violation';
    END IF;
    RETURN OLD;
END;
$$
LANGUAGE plpgsql;

--bun:split

DROP TRIGGER IF EXISTS "trg_ai_audit_events_append_only" ON "ai_audit_events";

--bun:split

CREATE TRIGGER "trg_ai_audit_events_append_only"
    BEFORE UPDATE OR DELETE ON "ai_audit_events"
    FOR EACH ROW
    EXECUTE FUNCTION "ai_audit_events_append_only"();

--bun:split

DROP TRIGGER IF EXISTS "trg_ai_audit_events_no_truncate" ON "ai_audit_events";

--bun:split

CREATE TRIGGER "trg_ai_audit_events_no_truncate"
    BEFORE TRUNCATE ON "ai_audit_events"
    FOR EACH STATEMENT
    EXECUTE FUNCTION "ai_audit_events_append_only"();

--bun:split

CREATE TABLE IF NOT EXISTS "ai_audit_chain_heads"(
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "last_seq" BIGINT NOT NULL DEFAULT 0,
    "last_hash" VARCHAR(64) NOT NULL DEFAULT '',
    "hash_key_id" VARCHAR(40),
    "last_verified_seq" BIGINT NOT NULL DEFAULT 0,
    "last_verified_at" BIGINT,
    "last_verification_status" VARCHAR(20),
    "last_verification_failed_seq" BIGINT,
    "last_verification_detail" TEXT,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_ai_audit_chain_heads" PRIMARY KEY ("organization_id", "business_unit_id"),
    CONSTRAINT "ck_ai_audit_chain_heads_last_verification_status" CHECK ("last_verification_status" IS NULL OR "last_verification_status" IN ('Verified', 'Mismatch', 'KeyMissing')),
    CONSTRAINT "ck_ai_audit_chain_heads_last_seq" CHECK ("last_seq" >= 0),
    CONSTRAINT "fk_ai_audit_chain_heads_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_audit_chain_heads_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

COMMENT ON TABLE "ai_audit_chain_heads" IS 'Where each tenant''s AI audit chain ends, locked by the projector to assign the next seq, and what its last verification found';

--bun:split

CREATE TABLE IF NOT EXISTS "ai_audit_seals"(
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "to_seq" BIGINT NOT NULL,
    "from_seq" BIGINT NOT NULL,
    "head_hash" VARCHAR(64) NOT NULL,
    "hash_key_id" VARCHAR(40),
    "row_count" INTEGER NOT NULL,
    "sealed_at" BIGINT NOT NULL,
    CONSTRAINT "pk_ai_audit_seals" PRIMARY KEY ("organization_id", "business_unit_id", "to_seq"),
    CONSTRAINT "ck_ai_audit_seals_range" CHECK ("from_seq" >= 1 AND "to_seq" >= "from_seq" AND "row_count" = "to_seq" - "from_seq" + 1),
    CONSTRAINT "fk_ai_audit_seals_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_audit_seals_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

COMMENT ON TABLE "ai_audit_seals" IS 'One checkpoint per projector batch per tenant; kept when the rows it covers are pruned, so the chain still verifies from it';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_seals_sealed" ON "ai_audit_seals"("organization_id", "business_unit_id", "sealed_at");

--bun:split

CREATE TABLE IF NOT EXISTS "ai_audit_exports"(
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "requested_by_user_id" VARCHAR(100) NOT NULL,
    "format" VARCHAR(10) NOT NULL,
    "filters" JSONB NOT NULL DEFAULT '{}'::jsonb,
    "range_from" BIGINT NOT NULL,
    "range_to" BIGINT NOT NULL,
    "snapshot_seq" BIGINT NOT NULL DEFAULT 0,
    "status" VARCHAR(20) NOT NULL DEFAULT 'Pending',
    "row_count" BIGINT NOT NULL DEFAULT 0,
    "byte_size" BIGINT NOT NULL DEFAULT 0,
    "sha256" VARCHAR(64),
    "artifact_key" VARCHAR(500),
    "artifact_expires_at" BIGINT,
    "chain_key_id" VARCHAR(40),
    "chain_first_seq" BIGINT,
    "chain_last_seq" BIGINT,
    "chain_complete" BOOLEAN NOT NULL DEFAULT FALSE,
    "workflow_id" VARCHAR(255),
    "error_message" TEXT,
    "started_at" BIGINT,
    "completed_at" BIGINT,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_ai_audit_exports" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "ck_ai_audit_exports_format" CHECK ("format" IN ('CSV', 'JSON')),
    CONSTRAINT "ck_ai_audit_exports_status" CHECK ("status" IN ('Pending', 'Running', 'Succeeded', 'Failed', 'Expired')),
    CONSTRAINT "ck_ai_audit_exports_range" CHECK ("range_to" >= "range_from"),
    CONSTRAINT "fk_ai_audit_exports_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_audit_exports_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

COMMENT ON TABLE "ai_audit_exports" IS 'Requests to write the AI audit trail to a CSV or JSON file, and the file each produced';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_exports_listing" ON "ai_audit_exports"("organization_id", "business_unit_id", "created_at" DESC, "id" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_audit_exports_expiry" ON "ai_audit_exports"("artifact_expires_at") WHERE "artifact_key" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "ai_audit_projector_state"(
    "source" VARCHAR(40) NOT NULL,
    "watermark_ts" BIGINT NOT NULL DEFAULT 0,
    "watermark_id" VARCHAR(100) NOT NULL DEFAULT '',
    "updated_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_ai_audit_projector_state" PRIMARY KEY ("source"),
    CONSTRAINT "ck_ai_audit_projector_state_source" CHECK ("source" IN ('agent_runs', 'assistant_turns', 'ai_usage_records', 'agent_run_steps', 'agent_run_events', 'agent_proposals', 'agent_decisions'))
);

COMMENT ON TABLE "ai_audit_projector_state" IS 'How far the AI audit projector has read each source, by the timestamp column it scans and the last id read';
