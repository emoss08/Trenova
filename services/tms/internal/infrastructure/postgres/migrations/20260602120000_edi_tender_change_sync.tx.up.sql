CREATE TYPE edi_tender_recipient_kind_enum AS ENUM(
    'Internal',
    'External'
);

CREATE TYPE edi_tender_recipient_status_enum AS ENUM(
    'Active',
    'Closed'
);

CREATE TYPE edi_tender_recipient_baseline_status_enum AS ENUM(
    'Sent',
    'Accepted'
);

CREATE TYPE edi_tender_change_status_enum AS ENUM(
    'PendingReview',
    'Applied',
    'Rejected',
    'Queued',
    'Sent',
    'Failed',
    'Ignored',
    'Superseded'
);

CREATE TYPE edi_message_delivery_status_enum AS ENUM(
    'Queued',
    'Sending',
    'Sent',
    'Failed'
);

--bun:split
ALTER TABLE "edi_messages"
    ADD COLUMN IF NOT EXISTS "delivery_status" edi_message_delivery_status_enum,
    ADD COLUMN IF NOT EXISTS "delivery_remote_path" text,
    ADD COLUMN IF NOT EXISTS "delivery_attempts" bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "delivery_last_attempt_at" bigint,
    ADD COLUMN IF NOT EXISTS "delivery_sent_at" bigint,
    ADD COLUMN IF NOT EXISTS "delivery_last_error" text;

CREATE INDEX IF NOT EXISTS "idx_edi_messages_delivery"
    ON "edi_messages"("organization_id", "business_unit_id", "delivery_status", "generated_at" DESC)
    WHERE "delivery_status" IS NOT NULL;

--bun:split
CREATE TABLE IF NOT EXISTS "edi_tender_recipients"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "source_organization_id" varchar(100) NOT NULL,
    "source_business_unit_id" varchar(100) NOT NULL,
    "source_shipment_id" varchar(100) NOT NULL,
    "recipient_kind" edi_tender_recipient_kind_enum NOT NULL,
    "recipient_organization_id" varchar(100),
    "recipient_business_unit_id" varchar(100),
    "edi_partner_id" varchar(100),
    "partner_document_profile_id" varchar(100),
    "communication_profile_id" varchar(100),
    "original_transfer_id" varchar(100),
    "shipment_link_id" varchar(100),
    "original_message_id" varchar(100),
    "latest_baseline_payload" jsonb NOT NULL,
    "latest_baseline_hash" varchar(128) NOT NULL,
    "baseline_recorded_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "baseline_status" edi_tender_recipient_baseline_status_enum NOT NULL,
    "status" edi_tender_recipient_status_enum NOT NULL DEFAULT 'Active',
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_tender_recipients" PRIMARY KEY ("id", "business_unit_id"),
    CONSTRAINT "fk_edi_tender_recipients_source_shipment" FOREIGN KEY ("source_shipment_id", "source_business_unit_id", "source_organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_tender_recipients_unique_recipient"
    ON "edi_tender_recipients"(
        "source_shipment_id",
        "source_business_unit_id",
        "source_organization_id",
        "recipient_kind",
        COALESCE("recipient_organization_id", ''),
        COALESCE("recipient_business_unit_id", ''),
        COALESCE("edi_partner_id", ''),
        COALESCE("partner_document_profile_id", '')
    );

CREATE INDEX IF NOT EXISTS "idx_edi_tender_recipients_source"
    ON "edi_tender_recipients"("source_organization_id", "source_business_unit_id", "source_shipment_id", "status");

CREATE INDEX IF NOT EXISTS "idx_edi_tender_recipients_recipient_org"
    ON "edi_tender_recipients"("recipient_organization_id", "business_unit_id", "status")
    WHERE "recipient_organization_id" IS NOT NULL;

--bun:split
CREATE TABLE IF NOT EXISTS "edi_tender_changes"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "source_organization_id" varchar(100) NOT NULL,
    "source_business_unit_id" varchar(100) NOT NULL,
    "source_shipment_id" varchar(100) NOT NULL,
    "recipient_id" varchar(100) NOT NULL,
    "recipient_kind" edi_tender_recipient_kind_enum NOT NULL,
    "status" edi_tender_change_status_enum NOT NULL DEFAULT 'PendingReview',
    "change_type" varchar(100) NOT NULL,
    "idempotency_key" varchar(255) NOT NULL,
    "source_shipment_version" bigint NOT NULL,
    "previous_baseline_payload" jsonb NOT NULL,
    "new_tender_payload" jsonb NOT NULL,
    "previous_baseline_hash" varchar(128) NOT NULL,
    "new_payload_hash" varchar(128) NOT NULL,
    "diff_summary" jsonb NOT NULL DEFAULT '{}',
    "conflict_metadata" jsonb NOT NULL DEFAULT '{}',
    "internal_transfer_id" varchar(100),
    "shipment_link_id" varchar(100),
    "outbound_message_id" varchar(100),
    "reviewed_by_id" varchar(100),
    "reviewed_at" bigint,
    "applied_by_id" varchar(100),
    "applied_at" bigint,
    "failure_reason" text,
    "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("change_type", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE(enum_to_text("status"), '')), 'B') ||
        setweight(immutable_to_tsvector('simple', COALESCE("failure_reason", '')), 'C')
    ) STORED,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_tender_changes" PRIMARY KEY ("id", "business_unit_id"),
    CONSTRAINT "fk_edi_tender_changes_recipient" FOREIGN KEY ("recipient_id", "business_unit_id") REFERENCES "edi_tender_recipients"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_tender_changes_idempotency"
    ON "edi_tender_changes"("recipient_id", "source_shipment_version", "new_payload_hash", "change_type");

CREATE INDEX IF NOT EXISTS "idx_edi_tender_changes_actionable"
    ON "edi_tender_changes"("recipient_id", "status", "created_at" DESC)
    WHERE "status" IN ('PendingReview', 'Queued');

CREATE INDEX IF NOT EXISTS "idx_edi_tender_changes_source"
    ON "edi_tender_changes"("source_organization_id", "source_business_unit_id", "source_shipment_id", "created_at" DESC);

CREATE INDEX IF NOT EXISTS "idx_edi_tender_changes_search"
    ON "edi_tender_changes" USING gin("search_vector");

--bun:split
WITH schema_row AS (
    SELECT "id" FROM "edi_source_context_schemas"
    WHERE "id" = 'edisc_x12_204_out_load_tender_v1'
)
INSERT INTO "edi_source_context_fields"(
    "id",
    "schema_id",
    "path",
    "source_kind",
    "data_type",
    "repeated",
    "repeat_path",
    "parent_path",
    "display_name",
    "description",
    "status"
)
SELECT
    'ediscf_x12_204_purpose_code',
    schema_row."id",
    'shipment.purposeCode',
    'shipment',
    'string',
    FALSE,
    NULL,
    'shipment',
    'Purpose Code',
    'X12 load tender transaction set purpose code.',
    'Active'
FROM schema_row
ON CONFLICT ("schema_id", "path", COALESCE("repeat_path", '')) DO NOTHING;
