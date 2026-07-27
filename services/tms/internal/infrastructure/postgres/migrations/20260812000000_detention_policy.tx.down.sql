ALTER TABLE "additional_charges"
    DROP CONSTRAINT IF EXISTS "fk_additional_charges_detention_occurrence";

--bun:split
ALTER TABLE "additional_charges"
    DROP COLUMN IF EXISTS "detention_occurrence_id";

--bun:split
ALTER TABLE "customer_billing_profiles"
    ADD COLUMN IF NOT EXISTS "detention_billing_enabled" boolean NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "detention_free_minutes" smallint NOT NULL DEFAULT 120,
    ADD COLUMN IF NOT EXISTS "detention_rate_per_hour" numeric(8, 2),
    ADD COLUMN IF NOT EXISTS "count_detention_only_on_appointment_stops" boolean NOT NULL DEFAULT FALSE;

--bun:split
ALTER TABLE "shipment_controls"
    DROP CONSTRAINT IF EXISTS "fk_shipment_controls_default_detention_policy";

--bun:split
ALTER TABLE "shipment_controls"
    DROP COLUMN IF EXISTS "default_detention_policy_id",
    DROP COLUMN IF EXISTS "use_detention_policy_engine";

--bun:split
DROP TABLE IF EXISTS "detention_notices";

--bun:split
DROP TABLE IF EXISTS "detention_evidence";

--bun:split
DROP TABLE IF EXISTS "detention_occurrences";

--bun:split
DROP TABLE IF EXISTS "detention_policy_tiers";

--bun:split
DROP TABLE IF EXISTS "detention_policies";

--bun:split
DROP TYPE IF EXISTS "detention_notice_delivery_status_enum";

DROP TYPE IF EXISTS "detention_notice_channel_enum";

DROP TYPE IF EXISTS "detention_notice_kind_enum";

DROP TYPE IF EXISTS "detention_evidence_source_enum";

DROP TYPE IF EXISTS "detention_evidence_kind_enum";

DROP TYPE IF EXISTS "detention_waiver_reason_enum";

DROP TYPE IF EXISTS "detention_notification_status_enum";

DROP TYPE IF EXISTS "detention_occurrence_status_enum";

DROP TYPE IF EXISTS "detention_cap_kind_enum";

DROP TYPE IF EXISTS "detention_cap_scope_enum";

DROP TYPE IF EXISTS "detention_unnotified_behavior_enum";

DROP TYPE IF EXISTS "detention_notification_requirement_enum";

DROP TYPE IF EXISTS "detention_tier_rate_unit_enum";

DROP TYPE IF EXISTS "detention_rate_source_enum";

DROP TYPE IF EXISTS "detention_rounding_mode_enum";

DROP TYPE IF EXISTS "detention_late_arrival_rule_enum";

DROP TYPE IF EXISTS "detention_clock_start_basis_enum";

DROP TYPE IF EXISTS "detention_policy_status_enum";
