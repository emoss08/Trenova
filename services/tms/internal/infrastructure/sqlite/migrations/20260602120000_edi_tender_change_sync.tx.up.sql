-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260602120000_edi_tender_change_sync.tx.up.sql

ALTER TABLE "edi_messages" ADD COLUMN "delivery_status" TEXT;

--bun:split

ALTER TABLE "edi_messages" ADD COLUMN "delivery_remote_path" TEXT;

--bun:split

ALTER TABLE "edi_messages" ADD COLUMN "delivery_attempts" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "edi_messages" ADD COLUMN "delivery_last_attempt_at" INTEGER;

--bun:split

ALTER TABLE "edi_messages" ADD COLUMN "delivery_sent_at" INTEGER;

--bun:split

ALTER TABLE "edi_messages" ADD COLUMN "delivery_last_error" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_messages_delivery"
    ON "edi_messages" ("organization_id", "business_unit_id", "delivery_status", "generated_at" DESC)WHERE "delivery_status" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "edi_tender_recipients"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "source_organization_id" TEXT NOT NULL,
    "source_business_unit_id" TEXT NOT NULL,
    "source_shipment_id" TEXT NOT NULL,
    "recipient_kind" TEXT NOT NULL,
    "recipient_organization_id" TEXT,
    "recipient_business_unit_id" TEXT,
    "edi_partner_id" TEXT,
    "partner_document_profile_id" TEXT,
    "communication_profile_id" TEXT,
    "original_transfer_id" TEXT,
    "shipment_link_id" TEXT,
    "original_message_id" TEXT,
    "latest_baseline_payload" TEXT NOT NULL,
    "latest_baseline_hash" TEXT NOT NULL,
    "baseline_recorded_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "baseline_status" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_tender_recipients" PRIMARY KEY ("id", "business_unit_id"),
    CONSTRAINT "fk_edi_tender_recipients_source_shipment" FOREIGN KEY ("source_shipment_id", "source_business_unit_id", "source_organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_tender_recipients_unique_recipient"
    ON "edi_tender_recipients" (
        "source_shipment_id",
        "source_business_unit_id",
        "source_organization_id",
        "recipient_kind",
        COALESCE("recipient_organization_id", ''),
        COALESCE("recipient_business_unit_id", ''),
        COALESCE("edi_partner_id", ''),
        COALESCE("partner_document_profile_id", '')
    );

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_tender_recipients_source"
    ON "edi_tender_recipients" ("source_organization_id", "source_business_unit_id", "source_shipment_id", "status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_tender_recipients_recipient_org"
    ON "edi_tender_recipients" ("recipient_organization_id", "business_unit_id", "status")WHERE "recipient_organization_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "edi_tender_changes"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "source_organization_id" TEXT NOT NULL,
    "source_business_unit_id" TEXT NOT NULL,
    "source_shipment_id" TEXT NOT NULL,
    "recipient_id" TEXT NOT NULL,
    "recipient_kind" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'PendingReview',
    "change_type" TEXT NOT NULL,
    "idempotency_key" TEXT NOT NULL,
    "source_shipment_version" INTEGER NOT NULL,
    "previous_baseline_payload" TEXT NOT NULL,
    "new_tender_payload" TEXT NOT NULL,
    "previous_baseline_hash" TEXT NOT NULL,
    "new_payload_hash" TEXT NOT NULL,
    "diff_summary" TEXT NOT NULL DEFAULT '{}',
    "conflict_metadata" TEXT NOT NULL DEFAULT '{}',
    "internal_transfer_id" TEXT,
    "shipment_link_id" TEXT,
    "outbound_message_id" TEXT,
    "reviewed_by_id" TEXT,
    "reviewed_at" INTEGER,
    "applied_by_id" TEXT,
    "applied_at" INTEGER,
    "failure_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_tender_changes" PRIMARY KEY ("id", "business_unit_id"),
    CONSTRAINT "fk_edi_tender_changes_recipient" FOREIGN KEY ("recipient_id", "business_unit_id") REFERENCES "edi_tender_recipients"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_tender_changes_idempotency"
    ON "edi_tender_changes" ("recipient_id", "source_shipment_version", "new_payload_hash", "change_type");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_tender_changes_actionable"
    ON "edi_tender_changes" ("recipient_id", "status", "created_at" DESC)WHERE "status" IN ('PendingReview', 'Queued');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_tender_changes_source"
    ON "edi_tender_changes" ("source_organization_id", "source_business_unit_id", "source_shipment_id", "created_at" DESC);
