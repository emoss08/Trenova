CREATE TABLE IF NOT EXISTS "gl_accounts"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "account_type_id" varchar(100) NOT NULL,
    "parent_id" varchar(100),
    "account_code" varchar(20) NOT NULL,
    "name" varchar(200) NOT NULL,
    "description" text,
    "is_system" boolean NOT NULL DEFAULT FALSE,
    "allow_manual_je" boolean NOT NULL DEFAULT TRUE,
    "require_project" boolean NOT NULL DEFAULT FALSE,
    "current_balance" bigint NOT NULL DEFAULT 0,
    "debit_balance" bigint NOT NULL DEFAULT 0,
    "credit_balance" bigint NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_gl_accounts" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_gl_accounts_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_gl_accounts_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_gl_accounts_account_type" FOREIGN KEY ("account_type_id", "organization_id", "business_unit_id") REFERENCES "account_types"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_gl_accounts_parent" FOREIGN KEY ("parent_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "uq_gl_accounts_code" UNIQUE ("organization_id", "business_unit_id", "account_code"),
    CONSTRAINT "chk_gl_accounts_no_self_parent" CHECK ("id" != "parent_id")
);

CREATE INDEX IF NOT EXISTS idx_gl_accounts_bu_org ON "gl_accounts"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS idx_gl_accounts_created_updated ON "gl_accounts"("created_at", "updated_at");

CREATE INDEX IF NOT EXISTS idx_gl_accounts_status ON "gl_accounts"("status");

CREATE INDEX IF NOT EXISTS idx_gl_accounts_account_type ON "gl_accounts"("account_type_id");

CREATE INDEX IF NOT EXISTS idx_gl_accounts_parent ON "gl_accounts"("parent_id");

CREATE INDEX IF NOT EXISTS idx_gl_accounts_code ON "gl_accounts"("account_code");

COMMENT ON TABLE "gl_accounts" IS 'Stores General Ledger accounts for the chart of accounts';

COMMENT ON COLUMN "gl_accounts"."account_code" IS 'Unique account code (e.g., 1000, 1010, 4000)';

COMMENT ON COLUMN "gl_accounts"."parent_id" IS 'Parent account for hierarchical chart of accounts';

COMMENT ON COLUMN "gl_accounts"."allow_manual_je" IS 'Whether manual journal entries can be posted to this account';

COMMENT ON COLUMN "gl_accounts"."require_project" IS 'Whether transactions require a project/job assignment';

COMMENT ON COLUMN "gl_accounts"."current_balance" IS 'Current balance in cents (denormalized for performance)';

COMMENT ON COLUMN "gl_accounts"."debit_balance" IS 'Total debits in cents (denormalized for performance)';

COMMENT ON COLUMN "gl_accounts"."credit_balance" IS 'Total credits in cents (denormalized for performance)';

-- 1. Search Vector Column
ALTER TABLE "gl_accounts"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_gl_accounts_search_vector ON "gl_accounts" USING GIN(search_vector);

--bun:split
CREATE OR REPLACE FUNCTION gl_accounts_search_trigger()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('english', COALESCE(NEW.account_code, '')), 'A') || setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') || setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS gl_accounts_search_update ON "gl_accounts";

CREATE TRIGGER gl_accounts_search_update
    BEFORE INSERT OR UPDATE ON "gl_accounts"
    FOR EACH ROW
    EXECUTE FUNCTION gl_accounts_search_trigger();

--bun:split
UPDATE
    "gl_accounts"
SET
    search_vector = setweight(to_tsvector('english', COALESCE(account_code, '')), 'A') || setweight(to_tsvector('english', COALESCE(name, '')), 'A') || setweight(to_tsvector('english', COALESCE(description, '')), 'B');

--bun:split
ALTER TABLE "gl_accounts"
    ALTER COLUMN "status" SET STATISTICS 1000;

--bun:split
ALTER TABLE "gl_accounts"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "gl_accounts"
    ALTER COLUMN "business_unit_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "gl_accounts"
    ALTER COLUMN "account_type_id" SET STATISTICS 1000;

