CREATE TYPE report_format_enum AS ENUM(
    'Csv',
    'Excel'
);

CREATE TYPE report_delivery_method_enum AS ENUM(
    'Download',
    'Email'
);

CREATE TYPE report_status_enum AS ENUM(
    'Pending',
    'Processing',
    'Completed',
    'Failed'
);

CREATE TABLE IF NOT EXISTS "reports"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "resource_type" varchar(100) NOT NULL,
    "name" varchar(255) NOT NULL,
    "format" report_format_enum NOT NULL,
    "delivery_method" report_delivery_method_enum NOT NULL,
    "status" report_status_enum NOT NULL DEFAULT 'Pending',
    "filter_state" jsonb NOT NULL DEFAULT '{}' ::jsonb,
    "file_path" text,
    "file_size" bigint DEFAULT 0,
    "row_count" int DEFAULT 0,
    "error_message" text,
    "completed_at" bigint,
    "expires_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_reports" PRIMARY KEY ("id"),
    CONSTRAINT "fk_reports_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_reports_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_reports_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_reports_org_bu ON "reports"("organization_id", "business_unit_id");

CREATE INDEX IF NOT EXISTS idx_reports_user ON "reports"("user_id");

CREATE INDEX IF NOT EXISTS idx_reports_status ON "reports"("status");

CREATE INDEX IF NOT EXISTS idx_reports_resource_type ON "reports"("resource_type");

CREATE INDEX IF NOT EXISTS idx_reports_created_at ON "reports"("created_at");

CREATE INDEX IF NOT EXISTS idx_reports_expires_at ON "reports"("expires_at")
WHERE
    "expires_at" IS NOT NULL;

COMMENT ON TABLE "reports" IS 'Stores generated data export reports for users';

COMMENT ON COLUMN "reports"."filter_state" IS 'Stores the complete filter/sort state used to generate the report';

COMMENT ON COLUMN "reports"."resource_type" IS 'The type of resource being exported (e.g., customer, shipment, user)';

COMMENT ON COLUMN "reports"."expires_at" IS 'When the report file expires and should be deleted from storage';

ALTER TABLE "reports"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_reports_search_vector ON "reports" USING GIN(search_vector);

--bun:split
CREATE OR REPLACE FUNCTION reports_search_trigger()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') || setweight(to_tsvector('english', COALESCE(NEW.resource_type, '')), 'B') || setweight(to_tsvector('english', COALESCE(NEW.status::text, '')), 'C');
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS reports_search_update ON "reports";

CREATE TRIGGER reports_search_update
    BEFORE INSERT OR UPDATE ON "reports"
    FOR EACH ROW
    EXECUTE FUNCTION reports_search_trigger();

--bun:split
ALTER TABLE "reports"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "reports"
    ALTER COLUMN "business_unit_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "reports"
    ALTER COLUMN "user_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "reports"
    ALTER COLUMN "status" SET STATISTICS 1000;

--bun:split
ALTER TABLE "reports"
    ALTER COLUMN "resource_type" SET STATISTICS 1000;

