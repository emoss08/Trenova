-- A turn is one question and the reply it is still producing.
--
-- Until now a turn existed only as the HTTP request carrying it: it had no id
-- until it had finished and proposed something, at which point the proposal
-- recorder opened an agent run for it. That is too late for two things this
-- needs. A turn's events are published under its id while it runs, so the id
-- has to exist first. And a reader whose stream expired has to be told
-- something true about what happened -- which means a row that outlives the
-- events and says how it ended.
--
-- It is not an agent run. A run is a unit of agent work worth listing and
-- auditing; a turn is somebody asking what their load count is. Opening a run
-- for every one of those would bury the first in the second. The run_id below
-- still points at one when the turn proposed a change, which is the case where
-- there genuinely is something to decide.
CREATE TABLE IF NOT EXISTS "assistant_turns"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "thread_id" varchar(100) NOT NULL,
    -- user_id is who asked. A turn runs as them and nobody else, which is what
    -- the relay checks before it hands anybody a reply.
    "user_id" varchar(100) NOT NULL,
    "run_id" varchar(100),
    "workflow_id" varchar(255),
    "status" varchar(20) NOT NULL DEFAULT 'Pending',
    "error_message" text,
    "started_at" bigint,
    "completed_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_assistant_turns" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "ck_assistant_turns_status" CHECK ("status" IN ('Pending', 'Running', 'Completed', 'Refused', 'Stopped', 'Failed')),
    CONSTRAINT "fk_assistant_turns_thread" FOREIGN KEY ("thread_id", "business_unit_id", "organization_id") REFERENCES "assistant_threads"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_turns_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_turns_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- One live turn per conversation, enforced here rather than hoped for.
--
-- Two turns appending to one thread interleave their message sequence numbers,
-- and the transcript stops describing anything that happened. A second send
-- has to stop the first, which it can only do if starting one while another is
-- running is impossible rather than merely unlikely.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_assistant_turns_active" ON "assistant_turns"("organization_id", "thread_id")
WHERE "status" IN ('Pending', 'Running');

--bun:split
-- Reading a conversation's turns newest first, which is how the panel asks.
CREATE INDEX IF NOT EXISTS "idx_assistant_turns_thread" ON "assistant_turns"("organization_id", "business_unit_id", "thread_id", "created_at" DESC);
