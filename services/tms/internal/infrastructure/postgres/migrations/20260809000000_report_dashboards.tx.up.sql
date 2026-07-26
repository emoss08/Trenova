--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "report_dashboards"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(255) NOT NULL,
    "description" text,
    "category" varchar(100),
    "tags" text[],
    "owner_id" varchar(100) NOT NULL,
    "visibility" varchar(20) NOT NULL DEFAULT 'private',
    "layout" jsonb NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_report_dashboards" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_report_dashboards_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_dashboards_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_dashboards_owner" FOREIGN KEY ("owner_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "check_report_dashboards_visibility" CHECK ("visibility" IN ('private', 'shared')),
    CONSTRAINT "check_report_dashboards_layout_format" CHECK (jsonb_typeof(layout) = 'object')
);

--bun:split
CREATE INDEX "idx_report_dashboards_tenant" ON "report_dashboards"("organization_id", "business_unit_id");

--bun:split
CREATE INDEX "idx_report_dashboards_owner" ON "report_dashboards"("owner_id", "organization_id", "business_unit_id");

--bun:split
CREATE UNIQUE INDEX "idx_report_dashboards_name" ON "report_dashboards"(lower("name"), "owner_id", "organization_id", "business_unit_id");

--bun:split
COMMENT ON TABLE report_dashboards IS 'Dashboards: a grid of tiles pointing at saved or canned reports, driven by shared parameters';
