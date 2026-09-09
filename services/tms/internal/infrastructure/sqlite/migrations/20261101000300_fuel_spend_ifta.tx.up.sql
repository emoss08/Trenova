-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261101000300_fuel_spend_ifta.tx.up.sql

CREATE TABLE IF NOT EXISTS "fuel_cards"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "provider" TEXT NOT NULL,
    "last_four" TEXT NOT NULL,
    "label" TEXT NOT NULL,
    "external_card_id" TEXT,
    "assigned_worker_id" TEXT,
    "assigned_tractor_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "expires_at" INTEGER,
    "cancelled_at" INTEGER,
    "cancel_reason" TEXT,
    "notes" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_fuel_cards" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_fuel_cards_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_cards_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_cards_assigned_worker" FOREIGN KEY ("assigned_worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_fuel_cards_assigned_tractor" FOREIGN KEY ("assigned_tractor_id", "organization_id", "business_unit_id") REFERENCES "tractors"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_fuel_cards_cancelled" CHECK (("status" = 'Cancelled') = ("cancelled_at" IS NOT NULL)),
    CONSTRAINT "chk_fuel_cards_expires" CHECK ("expires_at" IS NULL OR "expires_at" > 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_fuel_cards_provider_last_four" ON "fuel_cards" ("organization_id", "business_unit_id", "provider", "last_four")WHERE
    "status" <> 'Cancelled';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fuel_cards_status" ON "fuel_cards" ("organization_id", "business_unit_id", "status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fuel_cards_assigned_tractor" ON "fuel_cards" ("organization_id", "business_unit_id", "assigned_tractor_id")WHERE
    "assigned_tractor_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fuel_cards_assigned_worker" ON "fuel_cards" ("organization_id", "business_unit_id", "assigned_worker_id")WHERE
    "assigned_worker_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "fuel_purchase_import_batches"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "provider" TEXT NOT NULL,
    "document_id" TEXT,
    "file_name" TEXT,
    "source_format" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "default_fuel_type" TEXT,
    "default_fuel_card_id" TEXT,
    "default_currency" TEXT NOT NULL DEFAULT 'USD',
    "mapping" TEXT,
    "unmapped_headers" TEXT,
    "summary" TEXT,
    "row_count" INTEGER NOT NULL DEFAULT 0,
    "error_count" INTEGER NOT NULL DEFAULT 0,
    "committed_count" INTEGER NOT NULL DEFAULT 0,
    "error" TEXT,
    "uploaded_by_id" TEXT,
    "staged_at" INTEGER,
    "committed_at" INTEGER,
    "committed_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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

CREATE INDEX IF NOT EXISTS "idx_fuel_purchase_import_batches_open" ON "fuel_purchase_import_batches" ("organization_id", "business_unit_id", "created_at" DESC)WHERE
    "status" IN ('Pending', 'Parsed');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fuel_purchase_import_batches_document" ON "fuel_purchase_import_batches" ("document_id")WHERE
    "document_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "fuel_purchases"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "tractor_id" TEXT NOT NULL,
    "worker_id" TEXT,
    "jurisdiction_id" TEXT NOT NULL,
    "fuel_card_id" TEXT,
    "card_last_four" TEXT,
    "purchased_at" INTEGER NOT NULL,
    "vendor" TEXT,
    "vendor_city" TEXT,
    "fuel_type" TEXT NOT NULL,
    "quantity" REAL NOT NULL,
    "quantity_unit" TEXT NOT NULL DEFAULT 'Gallon',
    "gallons" REAL NOT NULL,
    "unit_price" REAL,
    "total_amount_minor" INTEGER NOT NULL,
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "odometer" INTEGER,
    "transaction_reference" TEXT,
    "source" TEXT NOT NULL DEFAULT 'Manual',
    "import_batch_id" TEXT,
    "tax_paid" INTEGER NOT NULL DEFAULT 1,
    "notes" TEXT,
    "created_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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
    CONSTRAINT "chk_fuel_purchases_source" CHECK (("source" = 'CardImport') = ("import_batch_id" IS NOT NULL))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_fuel_purchases_transaction_reference" ON "fuel_purchases" ("organization_id", "business_unit_id", "transaction_reference")WHERE
    "transaction_reference" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fuel_purchases_purchased_at" ON "fuel_purchases" ("organization_id", "business_unit_id", "purchased_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fuel_purchases_tractor" ON "fuel_purchases" ("organization_id", "business_unit_id", "tractor_id", "purchased_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fuel_purchases_jurisdiction_fuel" ON "fuel_purchases" ("organization_id", "business_unit_id", "jurisdiction_id", "fuel_type", "purchased_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fuel_purchases_import_batch" ON "fuel_purchases" ("import_batch_id")WHERE
    "import_batch_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fuel_purchases_fuel_card" ON "fuel_purchases" ("organization_id", "business_unit_id", "fuel_card_id")WHERE
    "fuel_card_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "fuel_purchase_import_rows"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "import_batch_id" TEXT NOT NULL,
    "row_number" INTEGER NOT NULL,
    "cells" TEXT,
    "parsed" TEXT,
    "transaction_reference" TEXT,
    "status" TEXT NOT NULL DEFAULT 'New',
    "error" TEXT,
    "resolved_tractor_id" TEXT,
    "resolved_fuel_card_id" TEXT,
    "resolved_jurisdiction_id" TEXT,
    "resolution_notes" TEXT,
    "fuel_purchase_id" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_fuel_purchase_import_rows" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_fuel_purchase_import_rows_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_purchase_import_rows_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_purchase_import_rows_batch" FOREIGN KEY ("import_batch_id", "organization_id", "business_unit_id") REFERENCES "fuel_purchase_import_batches"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_purchase_import_rows_purchase" FOREIGN KEY ("fuel_purchase_id", "organization_id", "business_unit_id") REFERENCES "fuel_purchases"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_fuel_purchase_import_rows_row_number" CHECK ("row_number" >= 1),
    CONSTRAINT "chk_fuel_purchase_import_rows_error" CHECK ("status" <> 'Error' OR "error" IS NOT NULL)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_fuel_purchase_import_rows_row_number" ON "fuel_purchase_import_rows" ("organization_id", "business_unit_id", "import_batch_id", "row_number");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fuel_purchase_import_rows_status" ON "fuel_purchase_import_rows" ("organization_id", "business_unit_id", "import_batch_id", "status");

--bun:split

CREATE TABLE IF NOT EXISTS "ifta_jurisdiction_mileage_entries"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "tractor_id" TEXT NOT NULL,
    "jurisdiction_id" TEXT NOT NULL,
    "traveled_at" INTEGER NOT NULL,
    "year" INTEGER NOT NULL,
    "quarter" INTEGER NOT NULL,
    "miles" REAL NOT NULL,
    "loaded" INTEGER NOT NULL DEFAULT 1,
    "source" TEXT NOT NULL DEFAULT 'Manual',
    "shipment_move_id" TEXT,
    "notes" TEXT,
    "created_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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

CREATE INDEX IF NOT EXISTS "idx_ifta_jurisdiction_mileage_entries_period" ON "ifta_jurisdiction_mileage_entries" ("organization_id", "business_unit_id", "year", "quarter", "jurisdiction_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ifta_jurisdiction_mileage_entries_tractor" ON "ifta_jurisdiction_mileage_entries" ("organization_id", "business_unit_id", "tractor_id", "traveled_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ifta_jurisdiction_mileage_entries_move" ON "ifta_jurisdiction_mileage_entries" ("organization_id", "business_unit_id", "shipment_move_id")WHERE
    "shipment_move_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "ifta_returns"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "year" INTEGER NOT NULL,
    "quarter" INTEGER NOT NULL,
    "amendment_number" INTEGER NOT NULL DEFAULT 0,
    "amends_return_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "timezone" TEXT NOT NULL,
    "period_start" INTEGER NOT NULL,
    "period_end" INTEGER NOT NULL,
    "total_miles" REAL NOT NULL DEFAULT 0,
    "total_taxable_miles" REAL NOT NULL DEFAULT 0,
    "total_gallons" REAL NOT NULL DEFAULT 0,
    "total_tax_paid_gallons" REAL NOT NULL DEFAULT 0,
    "net_taxable_gallons" REAL NOT NULL DEFAULT 0,
    "tax_due_minor" INTEGER NOT NULL DEFAULT 0,
    "surcharge_due_minor" INTEGER NOT NULL DEFAULT 0,
    "net_due_minor" INTEGER NOT NULL DEFAULT 0,
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "fleet_mpg_by_fuel_type" TEXT,
    "unattributed_miles" REAL NOT NULL DEFAULT 0,
    "unattributed_move_count" INTEGER NOT NULL DEFAULT 0,
    "no_tractor_miles" REAL NOT NULL DEFAULT 0,
    "no_tractor_move_count" INTEGER NOT NULL DEFAULT 0,
    "problems" TEXT,
    "computed_at" INTEGER,
    "finalized_at" INTEGER,
    "finalized_by_id" TEXT,
    "filed_at" INTEGER,
    "filed_by_id" TEXT,
    "filing_reference" TEXT,
    "reopened_at" INTEGER,
    "reopened_by_id" TEXT,
    "reopen_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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

CREATE UNIQUE INDEX IF NOT EXISTS "uq_ifta_returns_period" ON "ifta_returns" ("organization_id", "business_unit_id", "year", "quarter", "amendment_number");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_ifta_returns_open" ON "ifta_returns" ("organization_id", "business_unit_id", "year", "quarter")WHERE
    "status" IN ('Draft', 'Finalized');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ifta_returns_period" ON "ifta_returns" ("organization_id", "business_unit_id", "year" DESC, "quarter" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "ifta_return_lines"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "return_id" TEXT NOT NULL,
    "jurisdiction_id" TEXT NOT NULL,
    "fuel_type" TEXT NOT NULL,
    "is_ifta_member" INTEGER NOT NULL,
    "total_miles" REAL NOT NULL DEFAULT 0,
    "taxable_miles" REAL NOT NULL DEFAULT 0,
    "route_miles" REAL NOT NULL DEFAULT 0,
    "manual_miles" REAL NOT NULL DEFAULT 0,
    "loaded_miles" REAL NOT NULL DEFAULT 0,
    "empty_miles" REAL NOT NULL DEFAULT 0,
    "tax_paid_gallons" REAL NOT NULL DEFAULT 0,
    "tax_paid_gallons_raw" REAL NOT NULL DEFAULT 0,
    "purchase_count" INTEGER NOT NULL DEFAULT 0,
    "taxable_gallons" REAL NOT NULL DEFAULT 0,
    "net_taxable_gallons" REAL NOT NULL DEFAULT 0,
    "rate_per_gallon" REAL,
    "surcharge_rate_per_gallon" REAL,
    "rate_missing" INTEGER NOT NULL DEFAULT 0,
    "tax_due_minor" INTEGER NOT NULL DEFAULT 0,
    "surcharge_due_minor" INTEGER NOT NULL DEFAULT 0,
    "line_total_minor" INTEGER NOT NULL DEFAULT 0,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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

CREATE UNIQUE INDEX IF NOT EXISTS "uq_ifta_return_lines_jurisdiction_fuel" ON "ifta_return_lines" ("organization_id", "business_unit_id", "return_id", "jurisdiction_id", "fuel_type");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ifta_return_lines_return" ON "ifta_return_lines" ("organization_id", "business_unit_id", "return_id", "sort_order");
