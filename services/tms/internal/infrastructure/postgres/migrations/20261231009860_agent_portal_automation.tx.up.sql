ALTER TABLE "agent_extensions"
    DROP CONSTRAINT IF EXISTS "ck_agent_extensions_type";

--bun:split
ALTER TABLE "agent_extensions"
    ADD CONSTRAINT "ck_agent_extensions_type" CHECK ("type" IN ('Exa', 'Parallel', 'BrowserUse'));

--bun:split
CREATE TABLE IF NOT EXISTS "agent_portal_sites" (
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "name" VARCHAR(100) NOT NULL,
    "purpose" VARCHAR(500) NOT NULL DEFAULT '',
    "domains" JSONB NOT NULL DEFAULT '[]'::jsonb,
    "start_url" VARCHAR(2048) NOT NULL,
    "credentials" JSONB NOT NULL DEFAULT '{}'::jsonb,
    "browser_profile_id" VARCHAR(100) NOT NULL DEFAULT '',
    "allow_actions" BOOLEAN NOT NULL DEFAULT FALSE,
    "enabled" BOOLEAN NOT NULL DEFAULT TRUE,
    "created_by_id" VARCHAR(100),
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    "updated_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_agent_portal_sites" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_agent_portal_sites_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_portal_sites_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_portal_sites_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_agent_portal_sites_domains" CHECK (jsonb_typeof("domains") = 'array')
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_portal_sites_name" ON "agent_portal_sites" ("organization_id", "business_unit_id", LOWER("name"));

--bun:split
COMMENT ON TABLE "agent_portal_sites" IS 'Third-party web portals an organization lets its agents read and act on through a portal automation extension: where the portal lives, how to sign in, and whether agents may act there or only read';

--bun:split
COMMENT ON COLUMN "agent_portal_sites"."domains" IS 'Hostnames the browser may visit for this site; navigation is confined to them and every page a task reached is checked against them';

--bun:split
COMMENT ON COLUMN "agent_portal_sites"."credentials" IS 'The sign-in, encrypted and bound to the tenant and the site; it is sent to the vendor as secrets and never shown to a model';

--bun:split
CREATE TABLE IF NOT EXISTS "agent_portal_runs" (
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "site_id" VARCHAR(100) NOT NULL,
    "extension_type" VARCHAR(50) NOT NULL,
    "mode" VARCHAR(10) NOT NULL,
    "status" VARCHAR(20) NOT NULL,
    "instruction" TEXT NOT NULL,
    "details" JSONB NOT NULL DEFAULT '{}'::jsonb,
    "used_sign_in" BOOLEAN NOT NULL DEFAULT FALSE,
    "vendor_task_id" VARCHAR(100) NOT NULL DEFAULT '',
    "vendor_session_id" VARCHAR(100) NOT NULL DEFAULT '',
    "answer" TEXT NOT NULL DEFAULT '',
    "visited_urls" JSONB NOT NULL DEFAULT '[]'::jsonb,
    "off_site_urls" JSONB NOT NULL DEFAULT '[]'::jsonb,
    "steps" INTEGER NOT NULL DEFAULT 0,
    "cost_usd" NUMERIC(19, 6) NOT NULL DEFAULT 0,
    "failure" TEXT NOT NULL DEFAULT '',
    "idempotency_key" VARCHAR(200),
    "agent_definition_id" VARCHAR(100),
    "agent_run_id" VARCHAR(100),
    "thread_id" VARCHAR(100),
    "proposal_id" VARCHAR(100),
    "requested_by_id" VARCHAR(100),
    "started_at" BIGINT NOT NULL,
    "finished_at" BIGINT,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    "updated_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_agent_portal_runs" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_agent_portal_runs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_portal_runs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_portal_runs_site" FOREIGN KEY ("site_id", "organization_id", "business_unit_id") REFERENCES "agent_portal_sites" ("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_agent_portal_runs_requested_by" FOREIGN KEY ("requested_by_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_agent_portal_runs_mode" CHECK ("mode" IN ('Read', 'Act')),
    CONSTRAINT "ck_agent_portal_runs_status" CHECK ("status" IN ('Running', 'Succeeded', 'Failed', 'Stopped', 'TimedOut', 'Strayed')),
    CONSTRAINT "ck_agent_portal_runs_cost" CHECK ("cost_usd" >= 0 AND "steps" >= 0)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_portal_runs_idempotency" ON "agent_portal_runs" ("organization_id", "business_unit_id", "idempotency_key") WHERE "idempotency_key" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_portal_runs_tenant_started" ON "agent_portal_runs" ("organization_id", "business_unit_id", "started_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_portal_runs_site" ON "agent_portal_runs" ("site_id", "organization_id", "business_unit_id");

--bun:split
COMMENT ON TABLE "agent_portal_runs" IS 'Every portal task an agent ran: the site, whether it only read or acted, the instruction as the browser received it, every page it reached, the outcome and its cost, and who asked from which run, conversation and proposal';

--bun:split
COMMENT ON COLUMN "agent_portal_runs"."off_site_urls" IS 'Pages the browser reached outside the site''s addresses; a run with any is Strayed and its answer is never given to a model';

--bun:split
COMMENT ON COLUMN "agent_portal_runs"."idempotency_key" IS 'The proposal an action ran from, so an approval retried continues the same task instead of acting twice';

--bun:split
ALTER TABLE "agent_waits"
    DROP CONSTRAINT IF EXISTS "ck_agent_waits_kind";

--bun:split
ALTER TABLE "agent_waits"
    ADD CONSTRAINT "ck_agent_waits_kind" CHECK ("kind" IN ('Time', 'StopArrival', 'StopDeparture', 'Reply', 'AppointmentNear', 'FreeTimeEnding', 'HOSDriveBelow', 'PortalRun'));

--bun:split
SELECT trenova_rls.reconcile();
