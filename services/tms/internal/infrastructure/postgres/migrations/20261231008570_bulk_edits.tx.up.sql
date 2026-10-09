--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "bulk_edits"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "resource" varchar(100) NOT NULL,
    "field" varchar(100) NOT NULL,
    "value" varchar(200) NOT NULL DEFAULT '',
    "selection" jsonb NOT NULL,
    "targets" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "status" varchar(20) NOT NULL,
    "total_count" integer NOT NULL DEFAULT 0,
    "changed_count" integer NOT NULL DEFAULT 0,
    "failed_count" integer NOT NULL DEFAULT 0,
    "processed_count" integer NOT NULL DEFAULT 0,
    "failure_message" text NOT NULL DEFAULT '',
    "completed_at" bigint,
    "undone_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_bulk_edits" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_bulk_edits_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_bulk_edits_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_bulk_edits_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_bulk_edits_status" CHECK ("status" IN ('Queued', 'Running', 'Completed', 'Failed', 'Undoing', 'Undone')),
    CONSTRAINT "ck_bulk_edits_selection" CHECK (jsonb_typeof("selection") = 'object'),
    CONSTRAINT "ck_bulk_edits_targets" CHECK (jsonb_typeof("targets") = 'array')
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_bulk_edits_user_created" ON "bulk_edits"("organization_id", "business_unit_id", "user_id", "created_at" DESC);

--bun:split
COMMENT ON TABLE "bulk_edits" IS 'One change a person made to many rows of a table at once: what was asked, each row''s value before it, and whether it was undone';

--bun:split
SELECT trenova_rls.reconcile();
