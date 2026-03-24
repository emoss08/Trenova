CREATE TABLE IF NOT EXISTS "business_units"
(
    "id"         varchar(100) NOT NULL,
    "name"       varchar(100) NOT NULL,
    "code"       varchar(10)  NOT NULL,
    "metadata"   jsonb                 DEFAULT '{}' ::jsonb,
    "version"    bigint       NOT NULL DEFAULT 0,
    "created_at" bigint       NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint       NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    PRIMARY KEY ("id"),
    CONSTRAINT "check_metadata_format" CHECK (jsonb_typeof(metadata) = 'object')
);

--bun:split
CREATE UNIQUE INDEX "idx_business_units_code" ON "business_units" (lower("code"));

COMMENT ON TABLE "business_units" IS 'Stores information about business units in a hierarchical structure';

