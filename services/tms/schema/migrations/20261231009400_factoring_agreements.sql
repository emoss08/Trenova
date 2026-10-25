--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- The terms an organization factors its invoices on, dated so a schedule always
-- uses the terms in force when it was submitted.
CREATE TABLE IF NOT EXISTS "factoring_agreements"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "factoring_company_id" varchar(100) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Draft',
    "effective_from" bigint NOT NULL,
    "effective_to" bigint,
    "advance_rate_bp" integer NOT NULL,
    "reserve_rate_bp" integer NOT NULL DEFAULT 0,
    "minimum_fee_minor" bigint NOT NULL DEFAULT 0,
    "chargeback_window_days" integer NOT NULL DEFAULT 90,
    "recourse" varchar(20) NOT NULL DEFAULT 'Recourse',
    "accounting_treatment" varchar(30) NOT NULL DEFAULT 'Sale',
    "funding_days" integer NOT NULL DEFAULT 2,
    "submission_time" integer NOT NULL DEFAULT 600,
    "notes" text,
    "activated_at" bigint,
    "ended_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_factoring_agreements" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_factoring_agreements_factor" FOREIGN KEY ("factoring_company_id", "organization_id", "business_unit_id") REFERENCES "factoring_companies"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "chk_factoring_agreements_status" CHECK ("status" IN ('Draft', 'Active', 'Ended')),
    CONSTRAINT "chk_factoring_agreements_recourse" CHECK ("recourse" IN ('Recourse', 'NonRecourse')),
    CONSTRAINT "chk_factoring_agreements_treatment" CHECK ("accounting_treatment" IN ('Sale', 'SecuredBorrowing')),
    CONSTRAINT "chk_factoring_agreements_advance" CHECK ("advance_rate_bp" BETWEEN 1 AND 10000),
    CONSTRAINT "chk_factoring_agreements_reserve" CHECK ("reserve_rate_bp" BETWEEN 0 AND 10000 AND "advance_rate_bp" + "reserve_rate_bp" <= 10000),
    CONSTRAINT "chk_factoring_agreements_minimum_fee" CHECK ("minimum_fee_minor" >= 0),
    CONSTRAINT "chk_factoring_agreements_window" CHECK ("chargeback_window_days" BETWEEN 0 AND 730),
    CONSTRAINT "chk_factoring_agreements_funding_days" CHECK ("funding_days" BETWEEN 0 AND 60),
    CONSTRAINT "chk_factoring_agreements_submission_time" CHECK ("submission_time" BETWEEN 0 AND 1439),
    CONSTRAINT "chk_factoring_agreements_dates" CHECK ("effective_to" IS NULL OR "effective_to" > "effective_from"),
    CONSTRAINT "chk_factoring_agreements_activated" CHECK (("status" = 'Draft') = ("activated_at" IS NULL)),
    CONSTRAINT "chk_factoring_agreements_ended" CHECK (("status" = 'Ended') = ("ended_at" IS NOT NULL)),
    CONSTRAINT "ex_factoring_agreements_active_overlap" EXCLUDE USING gist(
        "organization_id" WITH =,
        "business_unit_id" WITH =,
        "factoring_company_id" WITH =,
        int8range("effective_from", "effective_to", '[)') WITH &&
    ) WHERE ("status" = 'Active')
);

CREATE INDEX IF NOT EXISTS "idx_factoring_agreements_factor"
    ON "factoring_agreements"("organization_id", "business_unit_id", "factoring_company_id", "effective_from" DESC);

-- The fee by days outstanding: a rate, a flat amount, or both, per band. Bands
-- run from day zero without gaps; the last one may be open-ended.
CREATE TABLE IF NOT EXISTS "factoring_fee_tiers"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "agreement_id" varchar(100) NOT NULL,
    "days_from" integer NOT NULL,
    "days_to" integer,
    "rate_bp" integer NOT NULL DEFAULT 0,
    "flat_minor" bigint NOT NULL DEFAULT 0,
    CONSTRAINT "pk_factoring_fee_tiers" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_factoring_fee_tiers_agreement" FOREIGN KEY ("agreement_id", "organization_id", "business_unit_id") REFERENCES "factoring_agreements"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "chk_factoring_fee_tiers_days" CHECK ("days_from" >= 0 AND ("days_to" IS NULL OR "days_to" > "days_from")),
    CONSTRAINT "chk_factoring_fee_tiers_rate" CHECK ("rate_bp" BETWEEN 0 AND 10000),
    CONSTRAINT "chk_factoring_fee_tiers_flat" CHECK ("flat_minor" >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_factoring_fee_tiers_start"
    ON "factoring_fee_tiers"("organization_id", "business_unit_id", "agreement_id", "days_from");

-- Documents the factor wants in every packet beyond what the customer's billing
-- profile already requires.
CREATE TABLE IF NOT EXISTS "factoring_agreement_document_types"(
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "agreement_id" varchar(100) NOT NULL,
    "document_type_id" varchar(100) NOT NULL,
    CONSTRAINT "pk_factoring_agreement_document_types" PRIMARY KEY ("organization_id", "business_unit_id", "agreement_id", "document_type_id"),
    CONSTRAINT "fk_factoring_agreement_document_types_agreement" FOREIGN KEY ("agreement_id", "organization_id", "business_unit_id") REFERENCES "factoring_agreements"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_factoring_agreement_document_types_type" FOREIGN KEY ("document_type_id", "organization_id", "business_unit_id") REFERENCES "document_types"("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

-- A customer's notice that its invoices are payable to the factor. One open
-- notice per customer and factor; a released one stays as history.
CREATE TABLE IF NOT EXISTS "factoring_noas"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL,
    "factoring_company_id" varchar(100) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Draft',
    "sent_at" bigint,
    "sent_to" varchar(320),
    "acknowledged_at" bigint,
    "signed_document_id" varchar(100),
    "expires_at" bigint,
    "expiry_warned_at" bigint,
    "released_at" bigint,
    "release_reason" varchar(500),
    "notes" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_factoring_noas" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_factoring_noas_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_factoring_noas_factor" FOREIGN KEY ("factoring_company_id", "organization_id", "business_unit_id") REFERENCES "factoring_companies"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "chk_factoring_noas_status" CHECK ("status" IN ('Draft', 'Sent', 'Acknowledged', 'Released')),
    CONSTRAINT "chk_factoring_noas_sent" CHECK ("status" = 'Draft' OR "status" = 'Released' OR "sent_at" IS NOT NULL OR "acknowledged_at" IS NOT NULL),
    CONSTRAINT "chk_factoring_noas_acknowledged" CHECK ("status" <> 'Acknowledged' OR ("acknowledged_at" IS NOT NULL AND "signed_document_id" IS NOT NULL)),
    CONSTRAINT "chk_factoring_noas_released" CHECK (("status" = 'Released') = ("released_at" IS NOT NULL)),
    CONSTRAINT "chk_factoring_noas_expiry" CHECK ("expires_at" IS NULL OR "acknowledged_at" IS NULL OR "expires_at" > "acknowledged_at")
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_factoring_noas_open"
    ON "factoring_noas"("organization_id", "business_unit_id", "customer_id", "factoring_company_id")
    WHERE "status" <> 'Released';

CREATE INDEX IF NOT EXISTS "idx_factoring_noas_factor"
    ON "factoring_noas"("organization_id", "business_unit_id", "factoring_company_id", "status");

CREATE INDEX IF NOT EXISTS "idx_factoring_noas_expiring"
    ON "factoring_noas"("expires_at")
    WHERE "status" = 'Acknowledged' AND "expires_at" IS NOT NULL;

-- The last readiness verdict for each factored invoice: what the queue, the
-- page figures and Watchtower read. Rebuilt from the invoice, its shipments'
-- documents, its dispute and the customer's notice whenever one of them moves.
CREATE TABLE IF NOT EXISTS "factoring_invoice_states"(
    "invoice_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "factoring_company_id" varchar(100),
    "customer_id" varchar(100) NOT NULL,
    "verdict" varchar(20) NOT NULL,
    "checks" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "failed_checks" smallint NOT NULL DEFAULT 0,
    "open_balance_minor" bigint NOT NULL DEFAULT 0,
    "ready_since" bigint,
    "ready_announced_at" bigint,
    "evaluated_at" bigint NOT NULL,
    "created_at" bigint NOT NULL,
    CONSTRAINT "pk_factoring_invoice_states" PRIMARY KEY ("invoice_id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_factoring_invoice_states_invoice" FOREIGN KEY ("invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "chk_factoring_invoice_states_verdict" CHECK ("verdict" IN ('Ready', 'NotReady', 'Excluded')),
    CONSTRAINT "chk_factoring_invoice_states_ready_since" CHECK (("verdict" = 'Ready') = ("ready_since" IS NOT NULL)),
    CONSTRAINT "chk_factoring_invoice_states_failed" CHECK ("failed_checks" >= 0)
);

CREATE INDEX IF NOT EXISTS "idx_factoring_invoice_states_queue"
    ON "factoring_invoice_states"("organization_id", "business_unit_id", "verdict", "created_at" DESC, "invoice_id" DESC);

CREATE INDEX IF NOT EXISTS "idx_factoring_invoice_states_ready"
    ON "factoring_invoice_states"("ready_since")
    WHERE "verdict" = 'Ready';

ALTER TABLE "invoices"
    ADD COLUMN IF NOT EXISTS "factoring_override" varchar(20) NOT NULL DEFAULT 'Default';

ALTER TABLE "invoices"
    ADD COLUMN IF NOT EXISTS "factoring_override_reason" varchar(500);

ALTER TABLE "invoices"
    DROP CONSTRAINT IF EXISTS "chk_invoices_factoring_override_reason";

ALTER TABLE "invoices"
    ADD CONSTRAINT "chk_invoices_factoring_override_reason" CHECK ("factoring_override" = 'Exclude' OR "factoring_override_reason" IS NULL);

ALTER TABLE "invoices"
    DROP CONSTRAINT IF EXISTS "chk_invoices_factoring_override";

ALTER TABLE "invoices"
    ADD CONSTRAINT "chk_invoices_factoring_override" CHECK ("factoring_override" IN ('Default', 'Exclude'));

ALTER TABLE "invoices"
    ADD COLUMN IF NOT EXISTS "factoring_agreement_id" varchar(100);

ALTER TABLE "invoices"
    DROP CONSTRAINT IF EXISTS "fk_invoices_factoring_agreement";

ALTER TABLE "invoices"
    ADD CONSTRAINT "fk_invoices_factoring_agreement"
        FOREIGN KEY ("factoring_agreement_id", "organization_id", "business_unit_id")
        REFERENCES "factoring_agreements"("id", "organization_id", "business_unit_id")
        ON UPDATE NO ACTION ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS "idx_invoices_factoring_agreement"
    ON "invoices"("organization_id", "business_unit_id", "factoring_agreement_id")
    WHERE "factoring_agreement_id" IS NOT NULL;

SELECT trenova_rls.reconcile();
