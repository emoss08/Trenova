-- Where the connection's change feed has read up to, and how the last read
-- went. The cursor belongs to the provider adapter and is opaque here.
ALTER TABLE "accounting_connections"
    ADD COLUMN IF NOT EXISTS "change_cursor" text,
    ADD COLUMN IF NOT EXISTS "changes_read_at" bigint,
    ADD COLUMN IF NOT EXISTS "changes_error_category" varchar(30),
    ADD COLUMN IF NOT EXISTS "changes_error_message" text,
    ADD COLUMN IF NOT EXISTS "inbound_payment_policy" varchar(20) NOT NULL DEFAULT 'Propose';

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_inbound_payment_policy";

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_inbound_payment_policy" CHECK ("inbound_payment_policy" IN ('Off', 'Propose', 'Apply'));

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_changes_error_category";

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_changes_error_category" CHECK ("changes_error_category" IS NULL OR "changes_error_category" IN ('Transient', 'RateLimited', 'Auth', 'Validation', 'Mapping', 'ClosedPeriod', 'Currency', 'Duplicate', 'NotFound', 'Conflict', 'Configuration'));

--bun:split
-- Payments recorded in the accounting system against documents Trenova sent:
-- one row per provider payment, applied in Trenova, proposed to a person, or
-- ignored, per the connection's inbound payment policy.
CREATE TABLE IF NOT EXISTS "accounting_inbound_changes"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "connection_id" varchar(100) NOT NULL,
    "kind" varchar(30) NOT NULL,
    "external_id" varchar(100) NOT NULL,
    "external_number" varchar(100),
    "provider_modified_at" bigint,
    "provider_modified_by" varchar(200),
    "txn_date" bigint NOT NULL,
    "amount_minor" bigint NOT NULL,
    "currency_code" varchar(3) NOT NULL,
    "party_external_id" varchar(100),
    "party_name" varchar(200),
    "party_object_id" varchar(100),
    "document" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "status" varchar(20) NOT NULL DEFAULT 'Detected',
    "reason" varchar(30),
    "resolution" text,
    "applied_objects" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "decided_by_id" varchar(100),
    "decided_at" bigint,
    "note" text,
    "detected_at" bigint NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_accounting_inbound_changes" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_inbound_changes_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_inbound_changes_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_inbound_changes_connection" FOREIGN KEY ("connection_id", "business_unit_id", "organization_id") REFERENCES "accounting_connections"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_inbound_changes_decided_by" FOREIGN KEY ("decided_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_accounting_inbound_changes_kind" CHECK ("kind" IN ('CustomerPayment', 'BillPayment')),
    CONSTRAINT "ck_accounting_inbound_changes_status" CHECK ("status" IN ('Detected', 'Proposed', 'Applied', 'Ignored', 'Superseded')),
    CONSTRAINT "ck_accounting_inbound_changes_reason" CHECK ("reason" IS NULL OR "reason" IN ('PolicyPropose', 'PeriodNotOpen', 'UnknownDocument', 'PartyMismatch', 'Overpayment', 'PartialBillPayment', 'AlreadyPaid', 'CurrencyMismatch', 'Voided', 'NotTrenovaDocument', 'ApplyFailed')),
    CONSTRAINT "ck_accounting_inbound_changes_amount" CHECK ("amount_minor" >= 0),
    CONSTRAINT "ck_accounting_inbound_changes_decided" CHECK ("status" NOT IN ('Applied', 'Ignored') OR "decided_at" IS NOT NULL),
    CONSTRAINT "ck_accounting_inbound_changes_proposed" CHECK ("status" <> 'Proposed' OR "reason" IS NOT NULL)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_inbound_changes_external" ON "accounting_inbound_changes"("organization_id", "business_unit_id", "connection_id", "kind", "external_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_accounting_inbound_changes_status" ON "accounting_inbound_changes"("organization_id", "business_unit_id", "connection_id", "status", "detected_at");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_accounting_inbound_changes_detected" ON "accounting_inbound_changes"("detected_at")
    WHERE "status" = 'Detected';

--bun:split
-- Recognising a document Trenova itself sent, when the provider reports it back.
CREATE INDEX IF NOT EXISTS "idx_accounting_sync_records_external" ON "accounting_sync_records"("organization_id", "business_unit_id", "connection_id", "external_id")
    WHERE "external_id" IS NOT NULL;
