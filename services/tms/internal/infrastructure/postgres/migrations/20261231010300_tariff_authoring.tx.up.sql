ALTER TABLE "formula_templates"
    DROP COLUMN IF EXISTS "type",
    DROP COLUMN IF EXISTS "expression",
    DROP COLUMN IF EXISTS "schema_id",
    DROP COLUMN IF EXISTS "language_version",
    DROP COLUMN IF EXISTS "variable_definitions",
    DROP COLUMN IF EXISTS "breakdown_definitions",
    DROP COLUMN IF EXISTS "min_charge",
    DROP COLUMN IF EXISTS "max_charge",
    DROP COLUMN IF EXISTS "rounding_mode",
    DROP COLUMN IF EXISTS "rounding_precision",
    ADD COLUMN IF NOT EXISTS "source" text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "inputs" jsonb NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS "imports" jsonb NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS "test_count" integer NOT NULL DEFAULT 0;

--bun:split
ALTER TABLE "formula_templates"
    ALTER COLUMN "source" DROP DEFAULT,
    ADD CONSTRAINT "ck_formula_templates_test_count" CHECK ("test_count" >= 0),
    ADD CONSTRAINT "ck_formula_templates_inputs_array" CHECK (jsonb_typeof("inputs") = 'array'),
    ADD CONSTRAINT "ck_formula_templates_imports_array" CHECK (jsonb_typeof("imports") = 'array');

--bun:split
ALTER TABLE "formula_template_versions"
    DROP COLUMN IF EXISTS "type",
    DROP COLUMN IF EXISTS "expression",
    DROP COLUMN IF EXISTS "schema_id",
    DROP COLUMN IF EXISTS "language_version",
    DROP COLUMN IF EXISTS "variable_definitions",
    DROP COLUMN IF EXISTS "breakdown_definitions",
    DROP COLUMN IF EXISTS "min_charge",
    DROP COLUMN IF EXISTS "max_charge",
    DROP COLUMN IF EXISTS "rounding_mode",
    DROP COLUMN IF EXISTS "rounding_precision",
    ADD COLUMN IF NOT EXISTS "source" text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "inputs" jsonb NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS "imports" jsonb NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS "test_count" integer NOT NULL DEFAULT 0;

--bun:split
ALTER TABLE "formula_template_versions"
    ALTER COLUMN "source" DROP DEFAULT,
    ADD CONSTRAINT "ck_formula_template_versions_test_count" CHECK ("test_count" >= 0);

--bun:split
DROP TABLE IF EXISTS "formula_template_test_cases";

--bun:split
DROP TYPE IF EXISTS "formula_template_type_enum";

--bun:split
DROP TYPE IF EXISTS "rate_rounding_mode_enum";

--bun:split
CREATE TABLE IF NOT EXISTS "formula_libraries"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "name" varchar(64) NOT NULL,
    "description" text,
    "source" text NOT NULL,
    "imports" jsonb NOT NULL DEFAULT '[]',
    "test_count" integer NOT NULL DEFAULT 0,
    "status" formula_template_status_enum NOT NULL DEFAULT 'Draft',
    "submitted_by_id" varchar(100),
    "submitted_at" bigint,
    "approved_by_id" varchar(100),
    "approved_at" bigint,
    "review_comment" text,
    "version" bigint NOT NULL DEFAULT 0,
    "source_library_id" varchar(100),
    "source_version_number" bigint,
    "current_version_number" bigint NOT NULL DEFAULT 1,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_formula_libraries" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_formula_libraries_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_libraries_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_libraries_submitted_by" FOREIGN KEY ("submitted_by_id")
        REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_formula_libraries_approved_by" FOREIGN KEY ("approved_by_id")
        REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "ck_formula_libraries_name" CHECK ("name" ~ '^[a-z][A-Za-z0-9]*$'),
    CONSTRAINT "ck_formula_libraries_test_count" CHECK ("test_count" >= 0),
    CONSTRAINT "ck_formula_libraries_imports_array" CHECK (jsonb_typeof("imports") = 'array'),
    CONSTRAINT "ck_formula_libraries_current_version" CHECK ("current_version_number" >= 1)
);

--bun:split
ALTER TABLE "formula_libraries"
    ADD COLUMN IF NOT EXISTS "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('english', COALESCE("name", '')), 'A') ||
        setweight(immutable_to_tsvector('english', COALESCE("description", '')), 'B')
    ) STORED;

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uk_formula_libraries_name"
    ON "formula_libraries"("organization_id", "business_unit_id", lower("name"));

CREATE INDEX IF NOT EXISTS "idx_formula_libraries_tenant_created"
    ON "formula_libraries"("organization_id", "business_unit_id", "created_at" DESC);

CREATE INDEX IF NOT EXISTS "idx_formula_libraries_in_review_since"
    ON "formula_libraries"("submitted_at")
    WHERE "status" = 'InReview';

CREATE INDEX IF NOT EXISTS "idx_formula_libraries_search_vector"
    ON "formula_libraries" USING GIN("search_vector");

--bun:split
CREATE TABLE IF NOT EXISTS "formula_library_versions"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "library_id" varchar(100) NOT NULL,
    "version_number" bigint NOT NULL,
    "name" varchar(64) NOT NULL,
    "description" text,
    "source" text NOT NULL,
    "imports" jsonb NOT NULL DEFAULT '[]',
    "test_count" integer NOT NULL DEFAULT 0,
    "status" formula_template_status_enum NOT NULL,
    "change_message" text,
    "change_summary" jsonb,
    "created_by_id" varchar(100) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_formula_library_versions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "uk_formula_library_versions_number"
        UNIQUE ("library_id", "organization_id", "business_unit_id", "version_number"),
    CONSTRAINT "fk_formula_library_versions_library" FOREIGN KEY ("library_id", "organization_id", "business_unit_id")
        REFERENCES "formula_libraries"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_library_versions_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_library_versions_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "ck_formula_library_versions_number" CHECK ("version_number" >= 1),
    CONSTRAINT "ck_formula_library_versions_test_count" CHECK ("test_count" >= 0)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_formula_library_versions_history"
    ON "formula_library_versions"("organization_id", "business_unit_id", "library_id", "version_number" DESC);

CREATE INDEX IF NOT EXISTS "idx_formula_library_versions_approved"
    ON "formula_library_versions"("organization_id", "business_unit_id", "library_id", "version_number" DESC)
    WHERE "status" = 'Active';

--bun:split
CREATE TABLE IF NOT EXISTS "formula_library_reviews"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "library_id" varchar(100) NOT NULL,
    "round" smallint NOT NULL,
    "decision" formula_template_review_decision_enum NOT NULL,
    "actor_id" varchar(100),
    "comment" text,
    "base_version_number" bigint NOT NULL DEFAULT 0,
    "requested_reviewer_id" varchar(100),
    "reviewer_note" text,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_formula_library_reviews" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_formula_library_reviews_library" FOREIGN KEY ("library_id", "organization_id", "business_unit_id")
        REFERENCES "formula_libraries"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_library_reviews_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_library_reviews_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_library_reviews_actor" FOREIGN KEY ("actor_id")
        REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_formula_library_reviews_requested_reviewer" FOREIGN KEY ("requested_reviewer_id")
        REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "ck_formula_library_reviews_round" CHECK ("round" >= 1),
    CONSTRAINT "ck_formula_library_reviews_reviewer_note_length" CHECK (
        "reviewer_note" IS NULL OR char_length("reviewer_note") <= 4000
    )
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_formula_library_reviews_library"
    ON "formula_library_reviews"("organization_id", "business_unit_id", "library_id", "created_at" DESC);

--bun:split
CREATE TABLE IF NOT EXISTS "formula_template_version_imports"(
    "template_version_id" varchar(100) NOT NULL,
    "library_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "template_id" varchar(100) NOT NULL,
    "template_version" bigint NOT NULL,
    "library_version_number" bigint NOT NULL,
    "path" varchar(100) NOT NULL,
    CONSTRAINT "pk_formula_template_version_imports"
        PRIMARY KEY ("organization_id", "business_unit_id", "template_version_id", "library_id"),
    CONSTRAINT "fk_formula_template_version_imports_version" FOREIGN KEY ("template_version_id")
        REFERENCES "formula_template_versions"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_version_imports_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id")
        REFERENCES "formula_templates"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_version_imports_library_version"
        FOREIGN KEY ("library_id", "organization_id", "business_unit_id", "library_version_number")
        REFERENCES "formula_library_versions"("library_id", "organization_id", "business_unit_id", "version_number")
        ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_version_imports_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_version_imports_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_formula_template_version_imports_library"
    ON "formula_template_version_imports"("organization_id", "business_unit_id", "library_id", "library_version_number");

--bun:split
SELECT trenova_rls.reconcile();
