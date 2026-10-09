--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "case_checklist_templates"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "kind" varchar(30) NOT NULL,
    "customer_id" varchar(100),
    "items" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "updated_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_case_checklist_templates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_case_checklist_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_case_checklist_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_case_checklist_templates_customer" FOREIGN KEY ("customer_id", "business_unit_id", "organization_id") REFERENCES "customers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_case_checklist_templates_kind" CHECK ("kind" IN ('ReadyToBill', 'ReadyToClose')),
    CONSTRAINT "ck_case_checklist_templates_items" CHECK (jsonb_typeof("items") = 'array')
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_case_checklist_templates_organization" ON "case_checklist_templates"("organization_id", "business_unit_id", "kind") WHERE "customer_id" IS NULL;

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_case_checklist_templates_customer" ON "case_checklist_templates"("organization_id", "business_unit_id", "kind", "customer_id") WHERE "customer_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_case_checklist_templates_business_unit" ON "case_checklist_templates"("business_unit_id");

--bun:split
COMMENT ON TABLE "case_checklist_templates" IS 'How an organization, or one customer of it, wants a Desk case checklist laid out: the steps in order, whether each is required, optional or off, and the steps it added. A customer''s template replaces the organization''s for that customer';

--bun:split
CREATE TABLE IF NOT EXISTS "case_checklist_ticks"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "subject_type" varchar(50) NOT NULL,
    "subject_id" varchar(100) NOT NULL,
    "item_key" varchar(50) NOT NULL,
    "ticked_by_id" varchar(100) NOT NULL,
    "ticked_at" bigint NOT NULL,
    CONSTRAINT "pk_case_checklist_ticks" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_case_checklist_ticks_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_case_checklist_ticks_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_case_checklist_ticks_user" FOREIGN KEY ("ticked_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_case_checklist_ticks_subject_type" CHECK ("subject_type" IN ('Shipment', 'Invoice'))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_case_checklist_ticks_item" ON "case_checklist_ticks"("organization_id", "business_unit_id", "subject_type", "subject_id", "item_key");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_case_checklist_ticks_business_unit" ON "case_checklist_ticks"("business_unit_id");

--bun:split
COMMENT ON TABLE "case_checklist_ticks" IS 'A person''s tick on a step an organization added to a case checklist, kept on the record so everyone working a case about it sees the same';

--bun:split
SELECT trenova_rls.reconcile();
