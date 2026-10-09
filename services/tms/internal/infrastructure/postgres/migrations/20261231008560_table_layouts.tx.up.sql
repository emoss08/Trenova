--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "table_layouts"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "resource" varchar(100) NOT NULL,
    "layout" jsonb NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_table_layouts" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_table_layouts_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_table_layouts_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_table_layouts_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_table_layouts_layout" CHECK (jsonb_typeof("layout") = 'object')
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_table_layouts_user_resource" ON "table_layouts"("organization_id", "business_unit_id", "user_id", "resource");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_table_layouts_business_unit" ON "table_layouts"("business_unit_id");

--bun:split
COMMENT ON TABLE "table_layouts" IS 'The last arrangement of each data table a person used: columns shown, ordered, sized and pinned, density and colour rules';

--bun:split
SELECT trenova_rls.reconcile();
