--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "fuel_card_provider_enum" AS ENUM(
    'Comdata',
    'EFS',
    'WEX',
    'Other'
);

--bun:split
CREATE TYPE "fuel_card_status_enum" AS ENUM(
    'Active',
    'Suspended',
    'Cancelled'
);

--bun:split
CREATE TYPE "fuel_quantity_unit_enum" AS ENUM(
    'Gallon',
    'Litre'
);

--bun:split
CREATE TYPE "fuel_purchase_source_enum" AS ENUM(
    'Manual',
    'CardImport'
);

--bun:split
CREATE TYPE "fuel_import_status_enum" AS ENUM(
    'Pending',
    'Parsed',
    'Committed',
    'Failed',
    'Discarded'
);

--bun:split
CREATE TYPE "fuel_import_row_status_enum" AS ENUM(
    'New',
    'DuplicateInFile',
    'AlreadyImported',
    'Error',
    'Committed',
    'Skipped'
);

--bun:split
CREATE TYPE "fuel_import_format_enum" AS ENUM(
    'CSV',
    'XLSX'
);

--bun:split
CREATE TYPE "ifta_mileage_source_enum" AS ENUM(
    'Manual',
    'RouteCalculation',
    'Telematics'
);

--bun:split
CREATE TYPE "ifta_return_status_enum" AS ENUM(
    'Draft',
    'Finalized',
    'Filed'
);

--bun:split
CREATE TABLE IF NOT EXISTS "fuel_cards"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "provider" fuel_card_provider_enum NOT NULL,
    "last_four" varchar(4) NOT NULL,
    "label" varchar(100) NOT NULL,
    "external_card_id" varchar(100),
    "assigned_worker_id" varchar(100),
    "assigned_tractor_id" varchar(100),
    "status" fuel_card_status_enum NOT NULL DEFAULT 'Active',
    "expires_at" bigint,
    "cancelled_at" bigint,
    "cancel_reason" text,
    "notes" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_fuel_cards" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_fuel_cards_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_cards_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_cards_assigned_worker" FOREIGN KEY ("assigned_worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_fuel_cards_assigned_tractor" FOREIGN KEY ("assigned_tractor_id", "organization_id", "business_unit_id") REFERENCES "tractors"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_fuel_cards_last_four" CHECK ("last_four" ~ '^[0-9]{4}$'),
    CONSTRAINT "chk_fuel_cards_cancelled" CHECK (("status" = 'Cancelled') = ("cancelled_at" IS NOT NULL)),
    CONSTRAINT "chk_fuel_cards_expires" CHECK ("expires_at" IS NULL OR "expires_at" > 0)
);

--bun:split
ALTER TABLE "fuel_cards"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("label", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("last_four", '')), 'B') || setweight(immutable_to_tsvector('simple', COALESCE("external_card_id", '')), 'B')) STORED;

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_fuel_cards_provider_last_four" ON "fuel_cards"("organization_id", "business_unit_id", "provider", "last_four")
WHERE
    "status" <> 'Cancelled';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_cards_status" ON "fuel_cards"("organization_id", "business_unit_id", "status");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_cards_assigned_tractor" ON "fuel_cards"("organization_id", "business_unit_id", "assigned_tractor_id")
WHERE
    "assigned_tractor_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_cards_assigned_worker" ON "fuel_cards"("organization_id", "business_unit_id", "assigned_worker_id")
WHERE
    "assigned_worker_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_cards_search" ON "fuel_cards" USING GIN(search_vector);

--bun:split
COMMENT ON TABLE fuel_cards IS 'A fleet fuel card identified by provider and last four digits. The external id is the provider''s masked token, never a PAN. A cancelled card keeps its history and frees its last four for reissue.';

--bun:split
CREATE TABLE IF NOT EXISTS "fuel_purchase_import_batches"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "provider" fuel_card_provider_enum NOT NULL,
    "document_id" varchar(100),
    "file_name" varchar(255),
    "source_format" fuel_import_format_enum,
    "status" fuel_import_status_enum NOT NULL DEFAULT 'Pending',
    "default_fuel_type" ifta_fuel_type_enum,
    "default_fuel_card_id" varchar(100),
    "default_currency" varchar(3) NOT NULL DEFAULT 'USD',
    "mapping" jsonb,
    "unmapped_headers" jsonb,
    "summary" jsonb,
    "row_count" integer NOT NULL DEFAULT 0,
    "error_count" integer NOT NULL DEFAULT 0,
    "committed_count" integer NOT NULL DEFAULT 0,
    "error" text,
    "uploaded_by_id" varchar(100),
    "staged_at" bigint,
    "committed_at" bigint,
    "committed_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_fuel_purchase_import_batches" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_fuel_purchase_import_batches_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_purchase_import_batches_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_purchase_import_batches_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_fuel_purchase_import_batches_default_card" FOREIGN KEY ("default_fuel_card_id", "organization_id", "business_unit_id") REFERENCES "fuel_cards"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_fuel_purchase_import_batches_parsed" CHECK ("status" <> 'Parsed' OR "document_id" IS NOT NULL),
    CONSTRAINT "chk_fuel_purchase_import_batches_committed" CHECK ("status" <> 'Committed' OR ("committed_at" IS NOT NULL AND "committed_by_id" IS NOT NULL)),
    CONSTRAINT "chk_fuel_purchase_import_batches_counts" CHECK ("row_count" >= 0 AND "error_count" >= 0 AND "committed_count" >= 0),
    CONSTRAINT "chk_fuel_purchase_import_batches_currency" CHECK (length("default_currency") = 3)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_purchase_import_batches_open" ON "fuel_purchase_import_batches"("organization_id", "business_unit_id", "created_at" DESC)
WHERE
    "status" IN ('Pending', 'Parsed');

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_purchase_import_batches_document" ON "fuel_purchase_import_batches"("document_id")
WHERE
    "document_id" IS NOT NULL;

--bun:split
COMMENT ON TABLE fuel_purchase_import_batches IS 'One uploaded fuel card statement. The sheet is staged into rows and summarised as a dry run before anyone commits it, so a statement nobody read never lands as tax records.';

--bun:split
CREATE TABLE IF NOT EXISTS "fuel_purchases"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "tractor_id" varchar(100) NOT NULL,
    "worker_id" varchar(100),
    "jurisdiction_id" varchar(100) NOT NULL,
    "fuel_card_id" varchar(100),
    "card_last_four" varchar(4),
    "purchased_at" bigint NOT NULL,
    "vendor" varchar(150),
    "vendor_city" varchar(100),
    "fuel_type" ifta_fuel_type_enum NOT NULL,
    "quantity" numeric(12, 3) NOT NULL,
    "quantity_unit" fuel_quantity_unit_enum NOT NULL DEFAULT 'Gallon',
    "gallons" numeric(12, 3) NOT NULL,
    "unit_price" numeric(19, 4),
    "total_amount_minor" bigint NOT NULL,
    "currency_code" varchar(3) NOT NULL DEFAULT 'USD',
    "odometer" bigint,
    "transaction_reference" varchar(150),
    "source" fuel_purchase_source_enum NOT NULL DEFAULT 'Manual',
    "import_batch_id" varchar(100),
    "tax_paid" boolean NOT NULL DEFAULT TRUE,
    "notes" text,
    "created_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_fuel_purchases" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_fuel_purchases_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_purchases_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_purchases_tractor" FOREIGN KEY ("tractor_id", "organization_id", "business_unit_id") REFERENCES "tractors"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_fuel_purchases_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_fuel_purchases_jurisdiction" FOREIGN KEY ("jurisdiction_id") REFERENCES "ifta_jurisdictions"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_fuel_purchases_fuel_card" FOREIGN KEY ("fuel_card_id", "organization_id", "business_unit_id") REFERENCES "fuel_cards"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_fuel_purchases_import_batch" FOREIGN KEY ("import_batch_id", "organization_id", "business_unit_id") REFERENCES "fuel_purchase_import_batches"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_fuel_purchases_purchased_at" CHECK ("purchased_at" > 0),
    CONSTRAINT "chk_fuel_purchases_quantity" CHECK ("quantity" > 0),
    CONSTRAINT "chk_fuel_purchases_gallons" CHECK ("gallons" > 0),
    CONSTRAINT "chk_fuel_purchases_unit_price" CHECK ("unit_price" IS NULL OR "unit_price" >= 0),
    CONSTRAINT "chk_fuel_purchases_total_amount" CHECK ("total_amount_minor" >= 0),
    CONSTRAINT "chk_fuel_purchases_odometer" CHECK ("odometer" IS NULL OR "odometer" >= 0),
    CONSTRAINT "chk_fuel_purchases_currency" CHECK (length("currency_code") = 3),
    CONSTRAINT "chk_fuel_purchases_card_last_four" CHECK ("card_last_four" IS NULL OR "card_last_four" ~ '^[0-9]{4}$'),
    CONSTRAINT "chk_fuel_purchases_source" CHECK (("source" = 'CardImport') = ("import_batch_id" IS NOT NULL))
);

--bun:split
ALTER TABLE "fuel_purchases"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("vendor", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("transaction_reference", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("vendor_city", '')), 'B')) STORED;

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_fuel_purchases_transaction_reference" ON "fuel_purchases"("organization_id", "business_unit_id", "transaction_reference")
WHERE
    "transaction_reference" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_purchases_purchased_at" ON "fuel_purchases"("organization_id", "business_unit_id", "purchased_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_purchases_tractor" ON "fuel_purchases"("organization_id", "business_unit_id", "tractor_id", "purchased_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_purchases_jurisdiction_fuel" ON "fuel_purchases"("organization_id", "business_unit_id", "jurisdiction_id", "fuel_type", "purchased_at");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_purchases_import_batch" ON "fuel_purchases"("import_batch_id")
WHERE
    "import_batch_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_purchases_fuel_card" ON "fuel_purchases"("organization_id", "business_unit_id", "fuel_card_id")
WHERE
    "fuel_card_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_purchases_search" ON "fuel_purchases" USING GIN(search_vector);

--bun:split
COMMENT ON TABLE fuel_purchases IS 'One fuel transaction against a tractor in a jurisdiction. Quantity is kept as entered beside the normalised gallons; the amount is in minor units of the currency entered. Tax-paid gallons earn IFTA credit; untaxed bulk fuel still counts toward fleet MPG. Receipts attach through the generic document association.';

--bun:split
CREATE TABLE IF NOT EXISTS "fuel_purchase_import_rows"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "import_batch_id" varchar(100) NOT NULL,
    "row_number" integer NOT NULL,
    "cells" jsonb,
    "parsed" jsonb,
    "transaction_reference" varchar(150),
    "status" fuel_import_row_status_enum NOT NULL DEFAULT 'New',
    "error" text,
    "resolved_tractor_id" varchar(100),
    "resolved_fuel_card_id" varchar(100),
    "resolved_jurisdiction_id" varchar(100),
    "resolution_notes" jsonb,
    "fuel_purchase_id" varchar(100),
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_fuel_purchase_import_rows" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_fuel_purchase_import_rows_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_purchase_import_rows_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_purchase_import_rows_batch" FOREIGN KEY ("import_batch_id", "organization_id", "business_unit_id") REFERENCES "fuel_purchase_import_batches"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_purchase_import_rows_purchase" FOREIGN KEY ("fuel_purchase_id", "organization_id", "business_unit_id") REFERENCES "fuel_purchases"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_fuel_purchase_import_rows_row_number" CHECK ("row_number" >= 1),
    CONSTRAINT "chk_fuel_purchase_import_rows_error" CHECK ("status" <> 'Error' OR "error" IS NOT NULL)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_fuel_purchase_import_rows_row_number" ON "fuel_purchase_import_rows"("organization_id", "business_unit_id", "import_batch_id", "row_number");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_purchase_import_rows_status" ON "fuel_purchase_import_rows"("organization_id", "business_unit_id", "import_batch_id", "status");

--bun:split
COMMENT ON TABLE fuel_purchase_import_rows IS 'One line of an uploaded statement as it was read, beside the purchase it parsed to and the ids it resolved. Errors are shown next to the cells that caused them.';

--bun:split
CREATE TABLE IF NOT EXISTS "ifta_jurisdiction_mileage_entries"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "tractor_id" varchar(100) NOT NULL,
    "jurisdiction_id" varchar(100) NOT NULL,
    "traveled_at" bigint NOT NULL,
    "year" smallint NOT NULL,
    "quarter" smallint NOT NULL,
    "miles" numeric(12, 2) NOT NULL,
    "loaded" boolean NOT NULL DEFAULT TRUE,
    "source" ifta_mileage_source_enum NOT NULL DEFAULT 'Manual',
    "shipment_move_id" varchar(100),
    "notes" text,
    "created_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_ifta_jurisdiction_mileage_entries" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_ifta_jurisdiction_mileage_entries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ifta_jurisdiction_mileage_entries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ifta_jurisdiction_mileage_entries_tractor" FOREIGN KEY ("tractor_id", "organization_id", "business_unit_id") REFERENCES "tractors"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_ifta_jurisdiction_mileage_entries_jurisdiction" FOREIGN KEY ("jurisdiction_id") REFERENCES "ifta_jurisdictions"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_ifta_jurisdiction_mileage_entries_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_ifta_jurisdiction_mileage_entries_traveled_at" CHECK ("traveled_at" > 0),
    CONSTRAINT "chk_ifta_jurisdiction_mileage_entries_quarter" CHECK ("quarter" >= 1 AND "quarter" <= 4),
    CONSTRAINT "chk_ifta_jurisdiction_mileage_entries_miles" CHECK ("miles" > 0 AND "miles" <= 100000),
    CONSTRAINT "chk_ifta_jurisdiction_mileage_entries_source" CHECK ("source" = 'Manual' OR "shipment_move_id" IS NOT NULL)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ifta_jurisdiction_mileage_entries_period" ON "ifta_jurisdiction_mileage_entries"("organization_id", "business_unit_id", "year", "quarter", "jurisdiction_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ifta_jurisdiction_mileage_entries_tractor" ON "ifta_jurisdiction_mileage_entries"("organization_id", "business_unit_id", "tractor_id", "traveled_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ifta_jurisdiction_mileage_entries_move" ON "ifta_jurisdiction_mileage_entries"("organization_id", "business_unit_id", "shipment_move_id")
WHERE
    "shipment_move_id" IS NOT NULL;

--bun:split
COMMENT ON TABLE ifta_jurisdiction_mileage_entries IS 'Miles a tractor drove in one jurisdiction, entered by hand or copied from a route. Year and quarter are fixed in the organization timezone when the row is written. An entry that names a shipment move replaces that move''s routed jurisdiction rows so nothing is counted twice.';

--bun:split
CREATE TABLE IF NOT EXISTS "ifta_returns"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "year" smallint NOT NULL,
    "quarter" smallint NOT NULL,
    "amendment_number" smallint NOT NULL DEFAULT 0,
    "amends_return_id" varchar(100),
    "status" ifta_return_status_enum NOT NULL DEFAULT 'Draft',
    "timezone" varchar(100) NOT NULL,
    "period_start" bigint NOT NULL,
    "period_end" bigint NOT NULL,
    "total_miles" numeric(14, 2) NOT NULL DEFAULT 0,
    "total_taxable_miles" numeric(14, 2) NOT NULL DEFAULT 0,
    "total_gallons" numeric(14, 3) NOT NULL DEFAULT 0,
    "total_tax_paid_gallons" numeric(14, 3) NOT NULL DEFAULT 0,
    "net_taxable_gallons" numeric(14, 3) NOT NULL DEFAULT 0,
    "tax_due_minor" bigint NOT NULL DEFAULT 0,
    "surcharge_due_minor" bigint NOT NULL DEFAULT 0,
    "net_due_minor" bigint NOT NULL DEFAULT 0,
    "currency_code" varchar(3) NOT NULL DEFAULT 'USD',
    "fleet_mpg_by_fuel_type" jsonb,
    "unattributed_miles" numeric(14, 2) NOT NULL DEFAULT 0,
    "unattributed_move_count" integer NOT NULL DEFAULT 0,
    "no_tractor_miles" numeric(14, 2) NOT NULL DEFAULT 0,
    "no_tractor_move_count" integer NOT NULL DEFAULT 0,
    "problems" jsonb,
    "computed_at" bigint,
    "finalized_at" bigint,
    "finalized_by_id" varchar(100),
    "filed_at" bigint,
    "filed_by_id" varchar(100),
    "filing_reference" varchar(100),
    "reopened_at" bigint,
    "reopened_by_id" varchar(100),
    "reopen_reason" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_ifta_returns" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_ifta_returns_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ifta_returns_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ifta_returns_amends" FOREIGN KEY ("amends_return_id", "organization_id", "business_unit_id") REFERENCES "ifta_returns"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_ifta_returns_quarter" CHECK ("quarter" >= 1 AND "quarter" <= 4),
    CONSTRAINT "chk_ifta_returns_amendment" CHECK ("amendment_number" >= 0 AND (("amendment_number" > 0) = ("amends_return_id" IS NOT NULL))),
    CONSTRAINT "chk_ifta_returns_period" CHECK ("period_end" > "period_start"),
    CONSTRAINT "chk_ifta_returns_currency" CHECK (length("currency_code") = 3),
    CONSTRAINT "chk_ifta_returns_finalized" CHECK (("status" IN ('Finalized', 'Filed')) = ("finalized_at" IS NOT NULL)),
    CONSTRAINT "chk_ifta_returns_filed" CHECK (("status" = 'Filed') = ("filed_at" IS NOT NULL)),
    CONSTRAINT "chk_ifta_returns_counts" CHECK ("unattributed_move_count" >= 0 AND "no_tractor_move_count" >= 0)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_ifta_returns_period" ON "ifta_returns"("organization_id", "business_unit_id", "year", "quarter", "amendment_number");

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_ifta_returns_open" ON "ifta_returns"("organization_id", "business_unit_id", "year", "quarter")
WHERE
    "status" IN ('Draft', 'Finalized');

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ifta_returns_period" ON "ifta_returns"("organization_id", "business_unit_id", "year" DESC, "quarter" DESC);

--bun:split
COMMENT ON TABLE ifta_returns IS 'A quarterly IFTA return worksheet. At most one working copy (Draft or Finalized) exists per period; a Filed return is immutable and can only be amended into a new Draft with the next amendment number. Totals and lines are a snapshot frozen at finalize.';

--bun:split
CREATE TABLE IF NOT EXISTS "ifta_return_lines"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "return_id" varchar(100) NOT NULL,
    "jurisdiction_id" varchar(100) NOT NULL,
    "fuel_type" ifta_fuel_type_enum NOT NULL,
    "is_ifta_member" boolean NOT NULL,
    "total_miles" numeric(12, 0) NOT NULL DEFAULT 0,
    "taxable_miles" numeric(12, 0) NOT NULL DEFAULT 0,
    "route_miles" numeric(12, 2) NOT NULL DEFAULT 0,
    "manual_miles" numeric(12, 2) NOT NULL DEFAULT 0,
    "loaded_miles" numeric(12, 2) NOT NULL DEFAULT 0,
    "empty_miles" numeric(12, 2) NOT NULL DEFAULT 0,
    "tax_paid_gallons" numeric(12, 0) NOT NULL DEFAULT 0,
    "tax_paid_gallons_raw" numeric(12, 3) NOT NULL DEFAULT 0,
    "purchase_count" integer NOT NULL DEFAULT 0,
    "taxable_gallons" numeric(12, 0) NOT NULL DEFAULT 0,
    "net_taxable_gallons" numeric(12, 0) NOT NULL DEFAULT 0,
    "rate_per_gallon" numeric(10, 4),
    "surcharge_rate_per_gallon" numeric(10, 4),
    "rate_missing" boolean NOT NULL DEFAULT FALSE,
    "tax_due_minor" bigint NOT NULL DEFAULT 0,
    "surcharge_due_minor" bigint NOT NULL DEFAULT 0,
    "line_total_minor" bigint NOT NULL DEFAULT 0,
    "sort_order" integer NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_ifta_return_lines" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_ifta_return_lines_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ifta_return_lines_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ifta_return_lines_return" FOREIGN KEY ("return_id", "organization_id", "business_unit_id") REFERENCES "ifta_returns"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ifta_return_lines_jurisdiction" FOREIGN KEY ("jurisdiction_id") REFERENCES "ifta_jurisdictions"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_ifta_return_lines_rate" CHECK ("rate_missing" OR "rate_per_gallon" IS NOT NULL),
    CONSTRAINT "chk_ifta_return_lines_surcharge" CHECK ("surcharge_due_minor" >= 0),
    CONSTRAINT "chk_ifta_return_lines_total" CHECK ("line_total_minor" = "tax_due_minor" + "surcharge_due_minor"),
    CONSTRAINT "chk_ifta_return_lines_purchase_count" CHECK ("purchase_count" >= 0)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_ifta_return_lines_jurisdiction_fuel" ON "ifta_return_lines"("organization_id", "business_unit_id", "return_id", "jurisdiction_id", "fuel_type");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ifta_return_lines_return" ON "ifta_return_lines"("organization_id", "business_unit_id", "return_id", "sort_order");

--bun:split
COMMENT ON TABLE ifta_return_lines IS 'One jurisdiction and fuel type on a return, with the routed, manual, loaded and empty miles that explain the total. Lines are replaced wholesale on recompute and carry no version. A negative tax due is a credit; the surcharge is charged on gallons consumed and is never a credit.';
