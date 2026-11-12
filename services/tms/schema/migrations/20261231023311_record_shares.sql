-- Sharing is one resource whatever is shared. A share says who sent which
-- record to whom, with what note, and where in the app the link opens (the
-- tab or the Desk page the sharer was on). It replaces invoice_shares, which
-- could only hold invoices; its rows are carried over as shares of an invoice.
-- A share never grants access: the recipient opens the link only because
-- their role already lets them.
CREATE TABLE IF NOT EXISTS "record_shares"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "record_kind" varchar(60) NOT NULL,
    "record_id" varchar(100) NOT NULL,
    "shared_with_id" varchar(100) NOT NULL,
    "shared_by_id" varchar(100) NOT NULL,
    "note" text,
    "path" text NOT NULL,
    "share_count" integer NOT NULL DEFAULT 1,
    "first_shared_at" bigint NOT NULL,
    "last_shared_at" bigint NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_record_shares" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_record_shares_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_record_shares_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_record_shares_shared_with" FOREIGN KEY ("shared_with_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_record_shares_shared_by" FOREIGN KEY ("shared_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_record_shares_share_count" CHECK ("share_count" >= 1),
    -- An address inside the app, never one that leaves it: a share's link is
    -- joined to the app's own origin.
    CONSTRAINT "ck_record_shares_path" CHECK ("path" LIKE '/%' AND "path" NOT LIKE '//%' AND length("path") <= 1000)
);

COMMENT ON TABLE "record_shares" IS 'Who shared which record with whom: one row per record and recipient, counted and re-dated each time it is shared again. A share sends a link; it grants no access';

COMMENT ON COLUMN "record_shares"."record_kind" IS 'The kind of record shared, as the permission registry names it: invoice, shipment, worker';

COMMENT ON COLUMN "record_shares"."path" IS 'The address inside the app the share opens: the record''s page on the tab, or the Desk page, the sharer was on';

-- One row per record and recipient, and the read every share dialog makes:
-- who has this record been shared with.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_record_shares_recipient" ON "record_shares"("organization_id", "business_unit_id", "record_id", "shared_with_id");

-- What has been shared with one person.
CREATE INDEX IF NOT EXISTS "idx_record_shares_shared_with" ON "record_shares"("organization_id", "business_unit_id", "shared_with_id", "last_shared_at" DESC);

INSERT INTO "record_shares"("id", "business_unit_id", "organization_id", "record_kind", "record_id", "shared_with_id", "shared_by_id", "note", "path", "share_count", "first_shared_at", "last_shared_at", "created_at", "updated_at")
SELECT
    'rsh_' || substr("id", position('_' IN "id") + 1),
    "business_unit_id",
    "organization_id",
    'invoice',
    "invoice_id",
    "shared_with_id",
    "shared_by_id",
    "note",
    '/billing/invoices?item=' || "invoice_id" || CASE WHEN "tab" = 'overview' THEN ''
    ELSE
        '&tab=' || "tab"
    END,
    "share_count",
    "first_shared_at",
    "last_shared_at",
    "created_at",
    "updated_at"
FROM
    "invoice_shares"
ON CONFLICT
    DO NOTHING;

DROP TABLE IF EXISTS "invoice_shares";

-- The templates a share is told with are no longer an invoice's: their kinds
-- and their variables changed, so a template written for the old ones would
-- not render. The built-in ones take over until an organization writes its own.
DELETE FROM "document_templates"
WHERE "kind" IN ('invoice.share.email', 'notification.invoice_shared');
