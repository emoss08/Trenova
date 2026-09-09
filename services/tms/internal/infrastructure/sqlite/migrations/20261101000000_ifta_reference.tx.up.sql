-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261101000000_ifta_reference.tx.up.sql

CREATE TABLE IF NOT EXISTS "ifta_jurisdictions"(
    "id" TEXT NOT NULL,
    "country_code" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "us_state_id" TEXT,
    "is_ifta_member" INTEGER NOT NULL,
    "has_surcharge" INTEGER NOT NULL DEFAULT 0,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_ifta_jurisdictions" PRIMARY KEY ("id"),
    CONSTRAINT "uq_ifta_jurisdictions_code" UNIQUE ("country_code", "code"),
    CONSTRAINT "fk_ifta_jurisdictions_us_state" FOREIGN KEY ("us_state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_ifta_jurisdictions_country_code" CHECK ("country_code" = upper("country_code") AND length("country_code") = 2),
    CONSTRAINT "chk_ifta_jurisdictions_code" CHECK ("code" = upper("code") AND length("code") = 2)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ifta_jurisdictions_member" ON "ifta_jurisdictions" ("is_ifta_member", "sort_order");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ifta_jurisdictions_us_state" ON "ifta_jurisdictions" ("us_state_id")WHERE
    "us_state_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "ifta_tax_rates"(
    "id" TEXT NOT NULL,
    "jurisdiction_id" TEXT NOT NULL,
    "year" INTEGER NOT NULL,
    "quarter" INTEGER NOT NULL,
    "fuel_type" TEXT NOT NULL,
    "rate_per_gallon" REAL NOT NULL,
    "surcharge_rate_per_gallon" REAL,
    "source_note" TEXT,
    "source_url" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_ifta_tax_rates" PRIMARY KEY ("id"),
    CONSTRAINT "uq_ifta_tax_rates_period" UNIQUE ("jurisdiction_id", "year", "quarter", "fuel_type"),
    CONSTRAINT "fk_ifta_tax_rates_jurisdiction" FOREIGN KEY ("jurisdiction_id") REFERENCES "ifta_jurisdictions"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_ifta_tax_rates_quarter" CHECK ("quarter" >= 1 AND "quarter" <= 4),
    CONSTRAINT "chk_ifta_tax_rates_rate" CHECK ("rate_per_gallon" >= 0),
    CONSTRAINT "chk_ifta_tax_rates_surcharge" CHECK ("surcharge_rate_per_gallon" IS NULL OR "surcharge_rate_per_gallon" >= 0)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ifta_tax_rates_period" ON "ifta_tax_rates" ("year", "quarter");
