-- The morning read. One row per organization, role and local day: the
-- figures gathered deterministically, the sections laid out from them, and
-- the model's wording where it was accepted. A briefing exists from the
-- moment the facts are gathered, so opening the page while the prose is
-- still being written shows the numbers rather than nothing.
CREATE TABLE IF NOT EXISTS "assistant_briefings"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "role_key" varchar(20) NOT NULL,
    "user_id" varchar(100),
    "briefing_date" varchar(10) NOT NULL,
    "run_id" varchar(100),
    "status" varchar(20) NOT NULL DEFAULT 'Pending',
    "headline" varchar(240),
    "sections" jsonb NOT NULL DEFAULT '[]',
    "facts" jsonb NOT NULL DEFAULT '{}',
    "narrated" boolean NOT NULL DEFAULT FALSE,
    "failure_reason" text,
    "emailed_at" bigint,
    "read_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_assistant_briefings" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_assistant_briefings_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_briefings_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_briefings_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_assistant_briefings_status" CHECK ("status" IN ('Pending', 'Ready', 'Failed')),
    CONSTRAINT "ck_assistant_briefings_role" CHECK ("role_key" IN ('Dispatch', 'Billing', 'Compliance', 'Leadership', 'General'))
);

--bun:split
-- One briefing per organization, role, reader and day. A shared briefing
-- has no reader, and the expression keeps those unique too: a NULL user_id
-- would otherwise let the nightly job write a second copy every time it
-- retried.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_assistant_briefings_day" ON "assistant_briefings"("organization_id", "business_unit_id", "role_key", "briefing_date", COALESCE("user_id", ''));

--bun:split
-- Reading today's briefing, and paging back through the week.
CREATE INDEX IF NOT EXISTS "idx_assistant_briefings_recent" ON "assistant_briefings"("organization_id", "business_unit_id", "briefing_date" DESC, "role_key");

--bun:split
-- When the morning briefing is written, in the organization's own hours.
-- On by default: a company that has agents at all wants the page they
-- produce, and one that does not simply reads a short one.
ALTER TABLE "agent_controls"
    ADD COLUMN IF NOT EXISTS "briefing_enabled" boolean NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS "briefing_hour_local" integer NOT NULL DEFAULT 6;

--bun:split
ALTER TABLE "agent_controls"
    ADD CONSTRAINT "ck_agent_controls_briefing_hour" CHECK ("briefing_hour_local" BETWEEN 0 AND 23);
