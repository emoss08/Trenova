CREATE TABLE IF NOT EXISTS "agent_web_watches" (
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "extension_type" VARCHAR(50) NOT NULL,
    "vendor_monitor_id" VARCHAR(100) NOT NULL DEFAULT '',
    "query" TEXT NOT NULL,
    "purpose" TEXT NOT NULL DEFAULT '',
    "include_domains" JSONB NOT NULL DEFAULT '[]'::jsonb,
    "cadence" VARCHAR(20) NOT NULL,
    "status" VARCHAR(20) NOT NULL,
    "secrets" JSONB NOT NULL DEFAULT '{}'::jsonb,
    "ends_at" BIGINT NOT NULL,
    "last_checked_at" BIGINT,
    "last_run_at" BIGINT,
    "run_count" INTEGER NOT NULL DEFAULT 0,
    "finding_count" INTEGER NOT NULL DEFAULT 0,
    "cost_usd" NUMERIC(19, 6) NOT NULL DEFAULT 0,
    "failure" TEXT NOT NULL DEFAULT '',
    "agent_definition_id" VARCHAR(100),
    "agent_run_id" VARCHAR(100),
    "thread_id" VARCHAR(100),
    "requested_by_id" VARCHAR(100),
    "stopped_by_id" VARCHAR(100),
    "finished_at" BIGINT,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    "updated_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_agent_web_watches" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_agent_web_watches_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_web_watches_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_web_watches_requested_by" FOREIGN KEY ("requested_by_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_agent_web_watches_stopped_by" FOREIGN KEY ("stopped_by_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_agent_web_watches_cadence" CHECK ("cadence" IN ('SixHours', 'Daily', 'Weekly')),
    CONSTRAINT "ck_agent_web_watches_status" CHECK ("status" IN ('Active', 'Ended', 'Stopped', 'Failed')),
    CONSTRAINT "ck_agent_web_watches_counts" CHECK ("run_count" >= 0 AND "finding_count" >= 0 AND "cost_usd" >= 0)
);

CREATE INDEX IF NOT EXISTS "idx_agent_web_watches_tenant_created" ON "agent_web_watches" ("organization_id", "business_unit_id", "created_at" DESC);

CREATE INDEX IF NOT EXISTS "idx_agent_web_watches_tenant_active" ON "agent_web_watches" ("organization_id", "business_unit_id") WHERE "status" = 'Active';

CREATE INDEX IF NOT EXISTS "idx_agent_web_watches_thread" ON "agent_web_watches" ("thread_id", "organization_id", "business_unit_id") WHERE "thread_id" IS NOT NULL;

COMMENT ON TABLE "agent_web_watches" IS 'Searches an agent set the vendor to run on a schedule, each handing back only what earlier runs did not find, until it ends, is stopped or fails';

COMMENT ON COLUMN "agent_web_watches"."secrets" IS 'The vendor''s webhook secret, encrypted and bound to the tenant and the watch; it verifies the vendor''s deliveries and is never shown';

CREATE TABLE IF NOT EXISTS "agent_web_watch_findings" (
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "watch_id" VARCHAR(100) NOT NULL,
    "vendor_run_id" VARCHAR(100) NOT NULL,
    "summary" TEXT NOT NULL DEFAULT '',
    "results" JSONB NOT NULL DEFAULT '[]'::jsonb,
    "found_at" BIGINT NOT NULL,
    "delivered_at" BIGINT,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_agent_web_watch_findings" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_agent_web_watch_findings_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_web_watch_findings_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_web_watch_findings_watch" FOREIGN KEY ("watch_id", "organization_id", "business_unit_id") REFERENCES "agent_web_watches" ("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_web_watch_findings_run" ON "agent_web_watch_findings" ("organization_id", "business_unit_id", "watch_id", "vendor_run_id");

CREATE INDEX IF NOT EXISTS "idx_agent_web_watch_findings_watch_found" ON "agent_web_watch_findings" ("watch_id", "organization_id", "business_unit_id", "found_at" DESC);

CREATE INDEX IF NOT EXISTS "idx_agent_web_watch_findings_undelivered" ON "agent_web_watch_findings" ("watch_id", "organization_id", "business_unit_id") WHERE "delivered_at" IS NULL;

COMMENT ON TABLE "agent_web_watch_findings" IS 'Each scheduled run of a web watch that found something new: the vendor''s summary and the pages, written outside the organization, and when it was handed to the work waiting on the watch';

ALTER TABLE "agent_waits"
    DROP CONSTRAINT IF EXISTS "ck_agent_waits_kind";

ALTER TABLE "agent_waits"
    ADD CONSTRAINT "ck_agent_waits_kind" CHECK ("kind" IN ('Time', 'StopArrival', 'StopDeparture', 'Reply', 'AppointmentNear', 'FreeTimeEnding', 'HOSDriveBelow', 'PortalRun', 'DocumentRead', 'WebWatch'));

SELECT trenova_rls.reconcile();
