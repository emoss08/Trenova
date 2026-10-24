--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- A factor's remittance file, staged and matched before anything is booked.
CREATE TABLE IF NOT EXISTS "factoring_remittances"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "factoring_company_id" varchar(100) NOT NULL,
    "document_id" varchar(100),
    "file_name" varchar(255),
    "source" varchar(20) NOT NULL DEFAULT 'Upload',
    "inbound_message_id" varchar(100),
    "source_format" varchar(10),
    "status" varchar(30) NOT NULL DEFAULT 'Pending',
    "default_kind" varchar(30) NOT NULL DEFAULT 'Funding',
    "remitted_on" bigint,
    "currency_code" varchar(3) NOT NULL DEFAULT 'USD',
    "mapping" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "headers" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "unmapped_headers" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "line_count" integer NOT NULL DEFAULT 0,
    "matched_count" integer NOT NULL DEFAULT 0,
    "exception_count" integer NOT NULL DEFAULT 0,
    "applied_count" integer NOT NULL DEFAULT 0,
    "total_cash_minor" bigint NOT NULL DEFAULT 0,
    "error" varchar(1000),
    "staged_at" bigint,
    "applied_at" bigint,
    "applied_by_id" varchar(100),
    "discarded_at" bigint,
    "discard_reason" varchar(500),
    "created_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_factoring_remittances" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_factoring_remittances_factor" FOREIGN KEY ("factoring_company_id", "organization_id", "business_unit_id") REFERENCES "factoring_companies"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_factoring_remittances_inbound_message" FOREIGN KEY ("inbound_message_id", "business_unit_id", "organization_id") REFERENCES "inbound_messages"("id", "business_unit_id", "organization_id") ON DELETE SET NULL ("inbound_message_id"),
    CONSTRAINT "chk_factoring_remittances_source" CHECK ("source" IN ('Upload', 'Email')),
    CONSTRAINT "chk_factoring_remittances_source_format" CHECK ("source_format" IS NULL OR "source_format" IN ('Csv', 'Xlsx', 'Pdf')),
    CONSTRAINT "chk_factoring_remittances_status" CHECK ("status" IN ('Pending', 'NeedsMapping', 'Parsed', 'Failed', 'PartiallyApplied', 'Applied', 'Discarded')),
    CONSTRAINT "chk_factoring_remittances_default_kind" CHECK ("default_kind" IN ('Funding', 'Rejection', 'ReserveRelease', 'Chargeback', 'Other')),
    CONSTRAINT "chk_factoring_remittances_staged" CHECK ("status" IN ('Pending', 'Failed', 'Discarded') OR "document_id" IS NOT NULL),
    CONSTRAINT "chk_factoring_remittances_failed" CHECK ("status" <> 'Failed' OR "error" IS NOT NULL),
    CONSTRAINT "chk_factoring_remittances_applied" CHECK ("status" NOT IN ('PartiallyApplied', 'Applied') OR "applied_at" IS NOT NULL),
    CONSTRAINT "chk_factoring_remittances_discarded" CHECK ("status" <> 'Discarded' OR ("discarded_at" IS NOT NULL AND "discard_reason" IS NOT NULL)),
    CONSTRAINT "chk_factoring_remittances_counts" CHECK ("line_count" >= 0 AND "matched_count" >= 0 AND "exception_count" >= 0 AND "applied_count" >= 0)
);

CREATE INDEX IF NOT EXISTS "idx_factoring_remittances_factor"
    ON "factoring_remittances"("organization_id", "business_unit_id", "factoring_company_id", "created_at" DESC, "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_factoring_remittances_open"
    ON "factoring_remittances"("organization_id", "business_unit_id", "created_at")
    WHERE "status" IN ('NeedsMapping', 'Parsed', 'PartiallyApplied');

CREATE UNIQUE INDEX IF NOT EXISTS "uq_factoring_remittances_inbound_document"
    ON "factoring_remittances"("organization_id", "business_unit_id", "inbound_message_id", "document_id")
    WHERE "inbound_message_id" IS NOT NULL;

-- One row of the remittance as the factor wrote it, what it was read as, and
-- the open schedule item it was matched to.
CREATE TABLE IF NOT EXISTS "factoring_remittance_lines"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "remittance_id" varchar(100) NOT NULL,
    "row_number" integer NOT NULL,
    "cells" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "kind" varchar(30) NOT NULL DEFAULT 'Funding',
    "invoice_number" varchar(100),
    "normalized_invoice" varchar(100),
    "factor_reference" varchar(100),
    "debtor_name" varchar(255),
    "face_minor" bigint,
    "advance_minor" bigint,
    "fee_minor" bigint,
    "reserve_minor" bigint,
    "cash_minor" bigint,
    "occurred_on" bigint,
    "match_status" varchar(20) NOT NULL DEFAULT 'Unmatched',
    "matched_by" varchar(10),
    "submission_item_id" varchar(100),
    "score" integer NOT NULL DEFAULT 0,
    "reason" varchar(255),
    "suggestions" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "error" varchar(500),
    "ignored_reason" varchar(500),
    "reserve_entry_id" varchar(100),
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_factoring_remittance_lines" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_factoring_remittance_lines_remittance" FOREIGN KEY ("remittance_id", "organization_id", "business_unit_id") REFERENCES "factoring_remittances"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_factoring_remittance_lines_item" FOREIGN KEY ("submission_item_id", "organization_id", "business_unit_id") REFERENCES "factoring_submission_items"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_factoring_remittance_lines_entry" FOREIGN KEY ("reserve_entry_id", "organization_id", "business_unit_id") REFERENCES "factoring_reserve_entries"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "chk_factoring_remittance_lines_kind" CHECK ("kind" IN ('Funding', 'Rejection', 'ReserveRelease', 'Chargeback', 'Other')),
    CONSTRAINT "chk_factoring_remittance_lines_match_status" CHECK ("match_status" IN ('Matched', 'Suggested', 'Unmatched', 'Ignored', 'Error', 'Applied')),
    CONSTRAINT "chk_factoring_remittance_lines_matched_by" CHECK ("matched_by" IS NULL OR "matched_by" IN ('Auto', 'Person')),
    CONSTRAINT "chk_factoring_remittance_lines_matched" CHECK ("match_status" NOT IN ('Matched', 'Applied') OR ("submission_item_id" IS NOT NULL AND "matched_by" IS NOT NULL)),
    CONSTRAINT "chk_factoring_remittance_lines_applied" CHECK ("match_status" <> 'Applied' OR "kind" = 'Rejection' OR "reserve_entry_id" IS NOT NULL),
    CONSTRAINT "chk_factoring_remittance_lines_error" CHECK ("match_status" <> 'Error' OR "error" IS NOT NULL),
    CONSTRAINT "chk_factoring_remittance_lines_ignored" CHECK ("match_status" <> 'Ignored' OR "ignored_reason" IS NOT NULL),
    CONSTRAINT "chk_factoring_remittance_lines_score" CHECK ("score" BETWEEN 0 AND 100),
    CONSTRAINT "uq_factoring_remittance_lines_row" UNIQUE ("organization_id", "business_unit_id", "remittance_id", "row_number")
);

CREATE INDEX IF NOT EXISTS "idx_factoring_remittance_lines_remittance"
    ON "factoring_remittance_lines"("organization_id", "business_unit_id", "remittance_id", "match_status", "row_number");

CREATE INDEX IF NOT EXISTS "idx_factoring_remittance_lines_item"
    ON "factoring_remittance_lines"("organization_id", "business_unit_id", "submission_item_id")
    WHERE "submission_item_id" IS NOT NULL;

-- How a factor's remittance files are laid out, saved from the first one a
-- person mapped, so the factor's next file stages on its own.
CREATE TABLE IF NOT EXISTS "factoring_remittance_profiles"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "factoring_company_id" varchar(100) NOT NULL,
    "mapping" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "header_signature" varchar(64) NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_factoring_remittance_profiles" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_factoring_remittance_profiles_factor" FOREIGN KEY ("factoring_company_id", "organization_id", "business_unit_id") REFERENCES "factoring_companies"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "uq_factoring_remittance_profiles_factor" UNIQUE ("organization_id", "business_unit_id", "factoring_company_id")
);

ALTER TABLE "factoring_reserve_entries"
    ADD COLUMN IF NOT EXISTS "remittance_line_id" varchar(100);

CREATE INDEX IF NOT EXISTS "idx_factoring_reserve_entries_remittance_line"
    ON "factoring_reserve_entries"("organization_id", "business_unit_id", "remittance_line_id")
    WHERE "remittance_line_id" IS NOT NULL;

ALTER TABLE "inbound_messages"
    DROP CONSTRAINT IF EXISTS "ck_inbound_messages_classification";

ALTER TABLE "inbound_messages"
    ADD CONSTRAINT "ck_inbound_messages_classification" CHECK ("classification" IS NULL OR "classification" IN ('Tender', 'RateConfirmation', 'ProofOfDelivery', 'Invoice', 'StatusRequest', 'DetentionDispute', 'FactorRemittance', 'Other'));

ALTER TABLE "triage_rules"
    DROP CONSTRAINT IF EXISTS "ck_triage_rules_classification";

ALTER TABLE "triage_rules"
    ADD CONSTRAINT "ck_triage_rules_classification" CHECK ("classification" IN ('Tender', 'RateConfirmation', 'ProofOfDelivery', 'Invoice', 'StatusRequest', 'DetentionDispute', 'FactorRemittance', 'Other'));

ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_feature";

ALTER TABLE "ai_usage_records"
    ADD CONSTRAINT "ck_ai_usage_records_feature" CHECK ("feature" IN ('AgentTurn', 'AgentEvaluation', 'TableQuery', 'FormulaGenerate', 'FormulaExplain', 'ShipmentImportChat', 'DocumentIntelligenceRoute', 'DocumentIntelligenceExtract', 'AccountingMapping', 'AgentReflection', 'AgentDrafting', 'AgentProfile', 'FactoringRemittanceExtract'));

SELECT trenova_rls.reconcile();
