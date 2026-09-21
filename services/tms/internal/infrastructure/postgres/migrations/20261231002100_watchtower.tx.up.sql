-- The watchtower is the one feed of what needs a person: findings, agent
-- work waiting on a decision, failures, carrier changes, weather, EDI files
-- held back. Each row is a projection of a record elsewhere, keyed by its
-- kind and id, and resolves when that record does.
CREATE TABLE IF NOT EXISTS "watchtower_items"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "source_kind" varchar(40) NOT NULL,
    "source_id" varchar(200) NOT NULL,
    "severity" varchar(20) NOT NULL,
    "title" varchar(200) NOT NULL,
    "summary" text,
    "subject_type" varchar(50),
    "subject_id" varchar(100),
    "event_kind" varchar(100),
    "path" varchar(500),
    "occurred_at" bigint NOT NULL,
    "resolved_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_watchtower_items" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_watchtower_items_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_watchtower_items_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_watchtower_items_severity" CHECK ("severity" IN ('Info', 'Warning', 'Critical'))
);

--bun:split
-- One item per source record.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_watchtower_items_source" ON "watchtower_items"("organization_id", "business_unit_id", "source_kind", "source_id");

--bun:split
-- The feed reads open items newest first; the counts read the same rows.
CREATE INDEX IF NOT EXISTS "idx_watchtower_items_open" ON "watchtower_items"("organization_id", "business_unit_id", "occurred_at" DESC, "id" DESC)
WHERE "resolved_at" IS NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_watchtower_items_feed" ON "watchtower_items"("organization_id", "business_unit_id", "occurred_at" DESC, "id" DESC);

--bun:split
-- Retention sweeps resolved rows by when they resolved.
CREATE INDEX IF NOT EXISTS "idx_watchtower_items_resolved" ON "watchtower_items"("resolved_at")
WHERE "resolved_at" IS NOT NULL;

--bun:split
-- Where each reader's eye last was: items after seen_at are new to them.
CREATE TABLE IF NOT EXISTS "watchtower_cursors"(
    "user_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "seen_at" bigint NOT NULL DEFAULT 0,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_watchtower_cursors" PRIMARY KEY ("user_id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_watchtower_cursors_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_watchtower_cursors_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_watchtower_cursors_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
