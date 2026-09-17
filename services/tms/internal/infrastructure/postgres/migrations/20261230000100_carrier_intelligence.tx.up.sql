CREATE TABLE IF NOT EXISTS "carrier_intel_controls"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "primary_provider" integration_type,
    "fallback_provider" integration_type,
    "enrollment_policy" varchar(20) NOT NULL DEFAULT 'Manual',
    "recent_usage_days" integer NOT NULL DEFAULT 90,
    "include_open_tenders" boolean NOT NULL DEFAULT FALSE,
    "auto_enroll_on_create" boolean NOT NULL DEFAULT FALSE,
    "auto_unenroll_on_inactive" boolean NOT NULL DEFAULT FALSE,
    "exclusive_watchlist" boolean NOT NULL DEFAULT FALSE,
    "poll_interval_minutes" integer NOT NULL DEFAULT 120,
    "snapshot_ttl_hours" integer NOT NULL DEFAULT 24,
    "full_profile_ttl_days" integer NOT NULL DEFAULT 30,
    "pretender_refresh_enabled" boolean NOT NULL DEFAULT FALSE,
    "pretender_max_age_hours" integer NOT NULL DEFAULT 24,
    "hard_max_age_hours" integer NOT NULL DEFAULT 168,
    "confirm_blocking_changes" boolean NOT NULL DEFAULT FALSE,
    "outage_policy" varchar(20) NOT NULL DEFAULT 'FailOpen',
    "auto_disqualify_on_block" boolean NOT NULL DEFAULT FALSE,
    "auto_apply_safety_rating" boolean NOT NULL DEFAULT FALSE,
    "rules" jsonb NOT NULL DEFAULT '{}',
    "auto_sync_fields" jsonb NOT NULL DEFAULT '[]',
    "monthly_spend_cap" numeric(19, 4),
    "soft_cap_percent" integer NOT NULL DEFAULT 80,
    "daily_full_profile_cap" integer,
    "raw_retention_days" integer NOT NULL DEFAULT 90,
    "snapshot_history_limit" integer NOT NULL DEFAULT 12,
    "self_monitoring_enabled" boolean NOT NULL DEFAULT FALSE,
    "policy_version" bigint NOT NULL DEFAULT 1,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_carrier_intel_controls" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_intel_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_carrier_intel_controls_enrollment_policy" CHECK ("enrollment_policy" IN ('AllActive', 'RecentlyUsed', 'Manual')),
    CONSTRAINT "chk_carrier_intel_controls_outage_policy" CHECK ("outage_policy" IN ('FailOpen', 'FailClosed')),
    CONSTRAINT "chk_carrier_intel_controls_recent_usage_days" CHECK ("recent_usage_days" BETWEEN 7 AND 365),
    CONSTRAINT "chk_carrier_intel_controls_poll_interval" CHECK ("poll_interval_minutes" BETWEEN 60 AND 1440),
    CONSTRAINT "chk_carrier_intel_controls_soft_cap_percent" CHECK ("soft_cap_percent" BETWEEN 1 AND 100),
    CONSTRAINT "chk_carrier_intel_controls_raw_retention_days" CHECK ("raw_retention_days" BETWEEN 7 AND 730),
    CONSTRAINT "chk_carrier_intel_controls_history_limit" CHECK ("snapshot_history_limit" BETWEEN 1 AND 100),
    CONSTRAINT "chk_carrier_intel_controls_spend_cap" CHECK ("monthly_spend_cap" IS NULL OR "monthly_spend_cap" >= 0),
    CONSTRAINT "chk_carrier_intel_controls_fallback" CHECK ("fallback_provider" IS NULL OR "fallback_provider" = 'FMCSAQCMobile')
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_carrier_intel_controls_tenant" ON "carrier_intel_controls"("organization_id", "business_unit_id");

--bun:split
COMMENT ON TABLE "carrier_intel_controls" IS 'Per-tenant carrier intelligence policy: which provider is primary, how carriers are enrolled for monitoring, rule actions, freshness windows and vendor spend caps.';

--bun:split
CREATE TABLE IF NOT EXISTS "carrier_intel_raw_payloads"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "provider" integration_type NOT NULL,
    "endpoint" varchar(64) NOT NULL,
    "dot_number" varchar(12),
    "payload" jsonb NOT NULL,
    "byte_size" integer NOT NULL DEFAULT 0,
    "fetched_at" bigint NOT NULL,
    "expires_at" bigint NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_carrier_intel_raw_payloads" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_intel_raw_payloads_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_raw_payloads_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_carrier_intel_raw_payloads_expiry" CHECK ("expires_at" > "fetched_at")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_intel_raw_payloads_expiry" ON "carrier_intel_raw_payloads"("organization_id", "business_unit_id", "expires_at");

--bun:split
COMMENT ON TABLE "carrier_intel_raw_payloads" IS 'Provider responses exactly as received, kept for audit and re-normalization. Contains contact data; purged at expires_at and never streamed or indexed.';

--bun:split
CREATE TABLE IF NOT EXISTS "carrier_intel_snapshots"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "subject_type" varchar(20) NOT NULL,
    "subject_id" varchar(100) NOT NULL,
    "carrier_id" varchar(100),
    "dot_number" varchar(12) NOT NULL,
    "docket_number" varchar(12),
    "provider" integration_type NOT NULL,
    "provider_ref" varchar(64),
    "depth" varchar(10) NOT NULL,
    "source" varchar(20) NOT NULL,
    "is_current" boolean NOT NULL DEFAULT FALSE,
    "not_found" boolean NOT NULL DEFAULT FALSE,
    "profile" jsonb NOT NULL,
    "findings" jsonb NOT NULL DEFAULT '[]',
    "blocking_codes" text[],
    "advisory_codes" text[],
    "risk_level" varchar(20) NOT NULL DEFAULT 'Unknown',
    "review_state" varchar(20) NOT NULL DEFAULT 'None',
    "reviewed_by_id" varchar(100),
    "reviewed_at" bigint,
    "review_note" text,
    "policy_version" bigint NOT NULL DEFAULT 0,
    "raw_payload_id" varchar(100),
    "content_hash" varchar(64) NOT NULL,
    "fetched_at" bigint NOT NULL,
    "source_as_of" bigint,
    "confirmed_at" bigint,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_carrier_intel_snapshots" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_intel_snapshots_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_snapshots_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_snapshots_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_snapshots_reviewed_by" FOREIGN KEY ("reviewed_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_carrier_intel_snapshots_subject_type" CHECK ("subject_type" IN ('Carrier', 'Organization', 'Customer', 'Prospect')),
    CONSTRAINT "chk_carrier_intel_snapshots_depth" CHECK ("depth" IN ('FMCSA', 'Lite', 'Full')),
    CONSTRAINT "chk_carrier_intel_snapshots_source" CHECK ("source" IN ('Primary', 'Fallback', 'ChangeFeedPatch')),
    CONSTRAINT "chk_carrier_intel_snapshots_review_state" CHECK ("review_state" IN ('None', 'NeedsReview', 'Reviewed'))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_carrier_intel_snapshots_current" ON "carrier_intel_snapshots"("organization_id", "business_unit_id", "subject_type", "subject_id")
WHERE
    "is_current";

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_intel_snapshots_subject_history" ON "carrier_intel_snapshots"("organization_id", "business_unit_id", "subject_type", "subject_id", "fetched_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_intel_snapshots_carrier" ON "carrier_intel_snapshots"("organization_id", "business_unit_id", "carrier_id")
WHERE
    "is_current" AND "carrier_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_intel_snapshots_dot" ON "carrier_intel_snapshots"("organization_id", "dot_number", "fetched_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_intel_snapshots_needs_review" ON "carrier_intel_snapshots"("organization_id", "business_unit_id", "fetched_at" DESC)
WHERE
    "is_current" AND "review_state" = 'NeedsReview';

--bun:split
COMMENT ON TABLE "carrier_intel_snapshots" IS 'Normalized carrier intelligence per subject. Exactly one row per subject is current; older rows are history pruned to the control''s snapshot_history_limit. Findings are the rule evaluation at policy_version.';

--bun:split
CREATE TABLE IF NOT EXISTS "carrier_intel_events"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "subject_type" varchar(20) NOT NULL,
    "subject_id" varchar(100) NOT NULL,
    "carrier_id" varchar(100),
    "dot_number" varchar(12) NOT NULL,
    "subject_name" varchar(255),
    "provider" integration_type NOT NULL,
    "source" varchar(30) NOT NULL,
    "category" varchar(30) NOT NULL,
    "field_path" varchar(200),
    "rule_code" varchar(100),
    "severity" varchar(20) NOT NULL,
    "action" varchar(10),
    "prior_value" jsonb,
    "current_value" jsonb,
    "summary" text NOT NULL,
    "vendor_changed_at" bigint,
    "detected_at" bigint NOT NULL,
    "fingerprint" varchar(64) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Open',
    "acknowledged_by_id" varchar(100),
    "acknowledged_at" bigint,
    "resolved_by_id" varchar(100),
    "resolved_at" bigint,
    "resolution" varchar(30),
    "resolution_note" text,
    "snapshot_id" varchar(100),
    "notified_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_carrier_intel_events" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_intel_events_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_events_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_events_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_events_acknowledged_by" FOREIGN KEY ("acknowledged_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_carrier_intel_events_resolved_by" FOREIGN KEY ("resolved_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_carrier_intel_events_status" CHECK ("status" IN ('Open', 'Acknowledged', 'Resolved', 'Dismissed')),
    CONSTRAINT "chk_carrier_intel_events_severity" CHECK ("severity" IN ('Critical', 'High', 'Medium', 'Low', 'Info')),
    CONSTRAINT "chk_carrier_intel_events_resolved" CHECK ("status" NOT IN ('Resolved', 'Dismissed') OR ("resolved_at" IS NOT NULL AND "resolution" IS NOT NULL))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_carrier_intel_events_fingerprint" ON "carrier_intel_events"("organization_id", "business_unit_id", "provider", "subject_type", "subject_id", "fingerprint");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_intel_events_inbox" ON "carrier_intel_events"("organization_id", "business_unit_id", "status", "detected_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_intel_events_carrier" ON "carrier_intel_events"("organization_id", "business_unit_id", "carrier_id", "detected_at" DESC)
WHERE
    "carrier_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_intel_events_subject" ON "carrier_intel_events"("organization_id", "business_unit_id", "subject_type", "subject_id", "detected_at" DESC);

--bun:split
COMMENT ON TABLE "carrier_intel_events" IS 'Detected carrier changes and rule hits awaiting acknowledgement. The fingerprint unique index makes overlapping change-feed polls idempotent.';

--bun:split
CREATE TABLE IF NOT EXISTS "carrier_monitoring_enrollments"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "subject_type" varchar(20) NOT NULL,
    "subject_id" varchar(100) NOT NULL,
    "carrier_id" varchar(100),
    "subject_name" varchar(255),
    "dot_number" varchar(12) NOT NULL,
    "docket_number" varchar(12),
    "provider" integration_type NOT NULL,
    "provider_ref" varchar(64),
    "mode" varchar(20) NOT NULL,
    "desired_state" varchar(20) NOT NULL,
    "vendor_state" varchar(20) NOT NULL DEFAULT 'Unknown',
    "reason" varchar(30) NOT NULL,
    "owned_by_trenova" boolean NOT NULL DEFAULT FALSE,
    "enrolled_at" bigint,
    "unenrolled_at" bigint,
    "last_synced_at" bigint,
    "last_confirmed_at" bigint,
    "last_used_at" bigint,
    "failure_count" integer NOT NULL DEFAULT 0,
    "last_error" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_carrier_monitoring_enrollments" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_monitoring_enrollments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_monitoring_enrollments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_monitoring_enrollments_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_carrier_monitoring_enrollments_mode" CHECK ("mode" IN ('Native', 'SnapshotDiff')),
    CONSTRAINT "chk_carrier_monitoring_enrollments_desired_state" CHECK ("desired_state" IN ('Enrolled', 'NotEnrolled')),
    CONSTRAINT "chk_carrier_monitoring_enrollments_vendor_state" CHECK ("vendor_state" IN ('Unknown', 'PendingAdd', 'Active', 'PendingRemove', 'Removed', 'Failed')),
    CONSTRAINT "chk_carrier_monitoring_enrollments_failure_count" CHECK ("failure_count" >= 0)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_carrier_monitoring_enrollments_subject" ON "carrier_monitoring_enrollments"("organization_id", "business_unit_id", "provider", "subject_type", "subject_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_monitoring_enrollments_pending" ON "carrier_monitoring_enrollments"("organization_id", "business_unit_id", "provider")
WHERE
    "vendor_state" IN ('Unknown', 'PendingAdd', 'PendingRemove') OR ("desired_state" = 'NotEnrolled' AND "vendor_state" = 'Active');

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_monitoring_enrollments_active" ON "carrier_monitoring_enrollments"("organization_id", "business_unit_id", "provider", "mode", "last_confirmed_at")
WHERE
    "desired_state" = 'Enrolled';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_monitoring_enrollments_carrier" ON "carrier_monitoring_enrollments"("organization_id", "business_unit_id", "carrier_id")
WHERE
    "carrier_id" IS NOT NULL;

--bun:split
COMMENT ON TABLE "carrier_monitoring_enrollments" IS 'Which subjects the organization wants watched and what the provider''s watchlist is known to hold. Reconciliation drives vendor_state toward desired_state.';

--bun:split
CREATE TABLE IF NOT EXISTS "carrier_intel_feed_states"(
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "provider" integration_type NOT NULL,
    "feed_type" varchar(32) NOT NULL,
    "cursor" jsonb NOT NULL DEFAULT '{}',
    "last_polled_at" bigint,
    "last_success_at" bigint,
    "next_poll_after" bigint,
    "paused_reason" varchar(30),
    "paused_at" bigint,
    "failure_count" integer NOT NULL DEFAULT 0,
    "last_error" text,
    CONSTRAINT "pk_carrier_intel_feed_states" PRIMARY KEY ("organization_id", "business_unit_id", "provider", "feed_type"),
    CONSTRAINT "fk_carrier_intel_feed_states_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_feed_states_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_carrier_intel_feed_states_failure_count" CHECK ("failure_count" >= 0),
    CONSTRAINT "chk_carrier_intel_feed_states_paused_reason" CHECK ("paused_reason" IS NULL OR "paused_reason" IN ('PaymentRequired', 'SpendCap', 'Unauthorized', 'Disabled'))
);

--bun:split
COMMENT ON TABLE "carrier_intel_feed_states" IS 'How far each tenant''s provider change feed has been read and whether polling is paused for billing, spend cap or credential problems.';

--bun:split
CREATE TABLE IF NOT EXISTS "carrier_equipment_verifications"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "carrier_assignment_id" varchar(100) NOT NULL,
    "shipment_move_id" varchar(100),
    "carrier_id" varchar(100) NOT NULL,
    "expected_dot_number" varchar(12),
    "unit_type" varchar(20) NOT NULL,
    "vin" varchar(17),
    "plate_number" varchar(15),
    "plate_state" varchar(2),
    "unit_number" varchar(50),
    "result" varchar(20) NOT NULL,
    "matched_dot_numbers" text[],
    "matched_legal_name" varchar(255),
    "detail" jsonb,
    "mismatch_reason" text,
    "provider" integration_type,
    "verified_by_id" varchar(100) NOT NULL,
    "verified_at" bigint NOT NULL,
    "override_by_id" varchar(100),
    "override_reason" text,
    "overridden_at" bigint,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_carrier_equipment_verifications" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_equipment_verifications_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_equipment_verifications_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_equipment_verifications_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_equipment_verifications_verified_by" FOREIGN KEY ("verified_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_carrier_equipment_verifications_override_by" FOREIGN KEY ("override_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_carrier_equipment_verifications_identifier" CHECK ("vin" IS NOT NULL OR ("plate_number" IS NOT NULL AND "plate_state" IS NOT NULL) OR "unit_number" IS NOT NULL),
    CONSTRAINT "chk_carrier_equipment_verifications_result" CHECK ("result" IN ('Match', 'Mismatch', 'NotFound', 'Unverifiable', 'ProviderError')),
    CONSTRAINT "chk_carrier_equipment_verifications_unit_type" CHECK ("unit_type" IN ('Tractor', 'Trailer', 'Straight')),
    CONSTRAINT "chk_carrier_equipment_verifications_override" CHECK ("overridden_at" IS NULL OR ("override_by_id" IS NOT NULL AND "override_reason" IS NOT NULL))
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_equipment_verifications_assignment" ON "carrier_equipment_verifications"("organization_id", "business_unit_id", "carrier_assignment_id", "verified_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_equipment_verifications_carrier" ON "carrier_equipment_verifications"("organization_id", "business_unit_id", "carrier_id", "verified_at" DESC);

--bun:split
COMMENT ON TABLE "carrier_equipment_verifications" IS 'Checks that the tractor or trailer arriving for a carrier-covered move is registered to that carrier, recorded with who checked and any override.';

--bun:split
CREATE TABLE IF NOT EXISTS "carrier_intel_usage_ledger"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "provider" integration_type NOT NULL,
    "endpoint" varchar(40) NOT NULL,
    "purpose" varchar(20) NOT NULL,
    "dot_number" varchar(12),
    "carrier_id" varchar(100),
    "billable" boolean NOT NULL DEFAULT FALSE,
    "billable_units" integer NOT NULL DEFAULT 0,
    "estimated_cost" numeric(19, 6) NOT NULL DEFAULT 0,
    "currency" varchar(3) NOT NULL DEFAULT 'USD',
    "dedupe_key" varchar(128),
    "status_code" integer NOT NULL DEFAULT 0,
    "outcome" varchar(20) NOT NULL,
    "latency_ms" integer NOT NULL DEFAULT 0,
    "initiated_by_type" varchar(20) NOT NULL DEFAULT 'System',
    "initiated_by_id" varchar(100),
    "workflow_id" varchar(200),
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_carrier_intel_usage_ledger" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_intel_usage_ledger_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_usage_ledger_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_carrier_intel_usage_ledger_cost" CHECK ("estimated_cost" >= 0 AND "billable_units" >= 0)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_intel_usage_ledger_tenant_created" ON "carrier_intel_usage_ledger"("organization_id", "business_unit_id", "created_at" DESC);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_carrier_intel_usage_ledger_dedupe" ON "carrier_intel_usage_ledger"("organization_id", "business_unit_id", "provider", "dedupe_key")
WHERE
    "billable" AND "dedupe_key" IS NOT NULL;

--bun:split
COMMENT ON TABLE "carrier_intel_usage_ledger" IS 'Every call made to a carrier intelligence provider with its estimated cost. A billable per-carrier-per-month charge carries a dedupe_key so repeat calls in the same month cost nothing more.';

--bun:split
CREATE TABLE IF NOT EXISTS "carrier_intel_usage_daily"(
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "provider" integration_type NOT NULL,
    "endpoint" varchar(40) NOT NULL,
    "day" integer NOT NULL,
    "calls" integer NOT NULL DEFAULT 0,
    "billable_units" integer NOT NULL DEFAULT 0,
    "estimated_cost" numeric(19, 6) NOT NULL DEFAULT 0,
    CONSTRAINT "pk_carrier_intel_usage_daily" PRIMARY KEY ("organization_id", "business_unit_id", "provider", "endpoint", "day"),
    CONSTRAINT "fk_carrier_intel_usage_daily_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_usage_daily_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
COMMENT ON TABLE "carrier_intel_usage_daily" IS 'Daily roll-up of the usage ledger, kept after ledger rows age out so spend history stays reportable.';

--bun:split
CREATE TABLE IF NOT EXISTS "carrier_intel_overrides"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "carrier_id" varchar(100) NOT NULL,
    "rule_code" varchar(100) NOT NULL,
    "reason" text NOT NULL,
    "granted_by_id" varchar(100) NOT NULL,
    "granted_at" bigint NOT NULL,
    "expires_at" bigint NOT NULL,
    "revoked_by_id" varchar(100),
    "revoked_at" bigint,
    "revoke_reason" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_carrier_intel_overrides" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_intel_overrides_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_overrides_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_overrides_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_intel_overrides_granted_by" FOREIGN KEY ("granted_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_carrier_intel_overrides_revoked_by" FOREIGN KEY ("revoked_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_carrier_intel_overrides_window" CHECK ("expires_at" > "granted_at" AND "expires_at" - "granted_at" <= 7776000),
    CONSTRAINT "chk_carrier_intel_overrides_revoked" CHECK ("revoked_at" IS NULL OR ("revoked_by_id" IS NOT NULL AND "revoke_reason" IS NOT NULL))
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_intel_overrides_active" ON "carrier_intel_overrides"("organization_id", "business_unit_id", "carrier_id", "expires_at")
WHERE
    "revoked_at" IS NULL;

--bun:split
COMMENT ON TABLE "carrier_intel_overrides" IS 'Time-boxed, audited exceptions that let a carrier be used despite a blocking intelligence finding. Capped at 90 days.';
