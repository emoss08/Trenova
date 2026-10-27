CREATE TYPE "tca_subscription_status_enum" AS ENUM ('Active', 'Paused');

CREATE TABLE IF NOT EXISTS "tca_allowlisted_tables" (
    "id"               VARCHAR(100) NOT NULL,
    "organization_id"  VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "table_name"       VARCHAR(100) NOT NULL,
    "display_name"     VARCHAR(255) NOT NULL,
    "enabled"          BOOLEAN NOT NULL DEFAULT true,
    "created_at"       BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at"       BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_tca_allowlist_org" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id"),
    CONSTRAINT "fk_tca_allowlist_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id"),
    UNIQUE ("organization_id", "business_unit_id", "table_name")
);

CREATE TABLE IF NOT EXISTS "tca_subscriptions" (
    "id"               VARCHAR(100) NOT NULL,
    "organization_id"  VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "user_id"          VARCHAR(100) NOT NULL,
    "name"             VARCHAR(255) NOT NULL,
    "table_name"       VARCHAR(100) NOT NULL,
    "record_id"        VARCHAR(100),
    "event_types"      JSONB NOT NULL DEFAULT '["INSERT","UPDATE","DELETE"]',
    "status"           tca_subscription_status_enum NOT NULL DEFAULT 'Active',
    "version"          BIGINT NOT NULL DEFAULT 0,
    "created_at"       BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at"       BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_tca_sub_org" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id"),
    CONSTRAINT "fk_tca_sub_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id"),
    CONSTRAINT "fk_tca_sub_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id")
);

CREATE INDEX "idx_tca_subscriptions_lookup"
    ON "tca_subscriptions" ("organization_id", "business_unit_id", "table_name", "status");

CREATE INDEX "idx_tca_subscriptions_user"
    ON "tca_subscriptions" ("organization_id", "business_unit_id", "user_id");
