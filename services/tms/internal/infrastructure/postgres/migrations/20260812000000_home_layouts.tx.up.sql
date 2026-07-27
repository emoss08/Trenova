--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "home_layout_presets"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(255) NOT NULL,
    "description" text,
    "layout" jsonb NOT NULL,
    "role_ids" text[],
    "core_responsibility" varchar(50),
    "is_org_default" boolean NOT NULL DEFAULT FALSE,
    "locked" boolean NOT NULL DEFAULT FALSE,
    "priority" integer NOT NULL DEFAULT 0,
    "created_by" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_home_layout_presets" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_home_layout_presets_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_home_layout_presets_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_home_layout_presets_created_by" FOREIGN KEY ("created_by") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "check_home_layout_presets_layout_format" CHECK (jsonb_typeof(layout) = 'object'),
    CONSTRAINT "check_home_layout_presets_priority" CHECK ("priority" BETWEEN 0 AND 1000),
    CONSTRAINT "check_home_layout_presets_core_responsibility" CHECK ("core_responsibility" IS NULL OR "core_responsibility" IN ('Billing', 'Operations', 'Finance', 'Leadership'))
);

--bun:split
CREATE INDEX "idx_home_layout_presets_tenant" ON "home_layout_presets"("organization_id", "business_unit_id");

--bun:split
CREATE UNIQUE INDEX "idx_home_layout_presets_name" ON "home_layout_presets"(lower("name"), "organization_id", "business_unit_id");

--bun:split
-- At most one preset per tenant may be the fallback every unassigned user lands on.
CREATE UNIQUE INDEX "idx_home_layout_presets_org_default" ON "home_layout_presets"("organization_id", "business_unit_id")
WHERE "is_org_default";

--bun:split
-- Resolution looks a preset up by the viewer's role IDs on every home screen load.
CREATE INDEX "idx_home_layout_presets_role_ids" ON "home_layout_presets" USING GIN("role_ids");

--bun:split
CREATE INDEX "idx_home_layout_presets_priority" ON "home_layout_presets"("organization_id", "business_unit_id", "priority" DESC);

--bun:split
COMMENT ON TABLE home_layout_presets IS 'Administrator-authored home screens assigned to roles, with an optional locked, org-default fallback';

--bun:split
CREATE TABLE IF NOT EXISTS "home_preferences"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "preferences" jsonb NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_home_preferences" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_home_preferences_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_home_preferences_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_home_preferences_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "check_home_preferences_format" CHECK (jsonb_typeof(preferences) = 'object')
);

--bun:split
CREATE UNIQUE INDEX "idx_home_preferences_user" ON "home_preferences"("user_id", "organization_id", "business_unit_id");

--bun:split
CREATE INDEX "idx_home_preferences_business_unit" ON "home_preferences"("business_unit_id");

--bun:split
CREATE INDEX "idx_home_preferences_organization" ON "home_preferences"("organization_id");

--bun:split
COMMENT ON TABLE home_preferences IS 'Stores each user''s home screen: which preset they follow, or the layout they customized';
