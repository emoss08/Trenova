CREATE TABLE IF NOT EXISTS "invoice_shares"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "invoice_id" varchar(100) NOT NULL,
    "shared_with_id" varchar(100) NOT NULL,
    "shared_by_id" varchar(100) NOT NULL,
    "note" text,
    "tab" varchar(20) NOT NULL DEFAULT 'overview',
    "share_count" integer NOT NULL DEFAULT 1,
    "first_shared_at" bigint NOT NULL,
    "last_shared_at" bigint NOT NULL,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_invoice_shares" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_invoice_shares_invoice" FOREIGN KEY ("invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_shares_shared_with" FOREIGN KEY ("shared_with_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_shares_shared_by" FOREIGN KEY ("shared_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_invoice_shares_tab" CHECK ("tab" IN ('overview', 'delivery', 'charges', 'documents', 'activity')),
    CONSTRAINT "chk_invoice_shares_share_count" CHECK ("share_count" >= 1)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_invoice_shares_recipient" ON "invoice_shares"("invoice_id", "organization_id", "business_unit_id", "shared_with_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_invoice_shares_shared_with" ON "invoice_shares"("shared_with_id", "organization_id", "business_unit_id");
