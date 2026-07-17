--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "sidebar_preferences"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "preferences" jsonb NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_sidebar_preferences" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_sidebar_preferences_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_sidebar_preferences_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_sidebar_preferences_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "check_sidebar_preferences_format" CHECK (jsonb_typeof(preferences) = 'object')
);

--bun:split
CREATE UNIQUE INDEX "idx_sidebar_preferences_user" ON "sidebar_preferences"("user_id", "organization_id", "business_unit_id");

--bun:split
CREATE INDEX "idx_sidebar_preferences_business_unit" ON "sidebar_preferences"("business_unit_id");

--bun:split
CREATE INDEX "idx_sidebar_preferences_organization" ON "sidebar_preferences"("organization_id");

--bun:split
COMMENT ON TABLE sidebar_preferences IS 'Stores per-user command-center sidebar customization preferences';
