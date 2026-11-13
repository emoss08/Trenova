--keel:notx
-- AI Control now says how each way of learning did over a stretch of days. That reads only the look
-- backs that called a model, which are a small part of a table where most rows were skipped.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_reflections_completed" ON "agent_reflections"("organization_id", "business_unit_id", "created_at" DESC) WHERE "status" = 'Completed';

--keel:split
-- A look back that could not be prepared was left Running, and a Running row read as a stretch of
-- the conversation already looked over, so what was said in it was never read. Preparing now marks
-- the row failed; these are the ones left from before, an hour being far longer than a look back
-- takes.
UPDATE "agent_reflections"
SET "status" = 'Failed',
    "error_message" = 'The look back was interrupted before it finished',
    "finished_at" = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint
WHERE "status" = 'Running'
  AND "created_at" < EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint - 3600;

--keel:split
-- An agent also looks back over what people keep changing, by hand, of the work it did: the same
-- field of the same kind of record changed on several records is how the organization wants it
-- set, and nobody had to say so. Such a look back is about a habit, not a conversation or a run,
-- so it is keyed by the habit and the day it was read.
ALTER TABLE "agent_reflections"
    ADD COLUMN IF NOT EXISTS "pattern_key" varchar(400);

COMMENT ON COLUMN "agent_reflections"."pattern_key" IS 'For a look back over what people changed of an agent''s work: the agent, kind of record and field they kept changing';

ALTER TABLE "agent_reflections"
    DROP CONSTRAINT IF EXISTS "ck_agent_reflections_subject_type";

ALTER TABLE "agent_reflections"
    ADD CONSTRAINT "ck_agent_reflections_subject_type" CHECK ("subject_type" IN ('Thread', 'Run', 'Correction'));

ALTER TABLE "agent_reflections"
    DROP CONSTRAINT IF EXISTS "ck_agent_reflections_subject";

ALTER TABLE "agent_reflections"
    ADD CONSTRAINT "ck_agent_reflections_subject" CHECK (("subject_type" <> 'Thread' OR "thread_id" IS NOT NULL) AND ("subject_type" <> 'Run' OR "run_id" IS NOT NULL) AND ("subject_type" <> 'Correction' OR "pattern_key" IS NOT NULL));

--keel:split
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS "uq_agent_reflections_correction" ON "agent_reflections"("organization_id", "business_unit_id", "pattern_key", "through_sequence") WHERE "subject_type" = 'Correction';

--keel:split
-- What agents wrote, newest first, is the first read of the nightly search for work people then
-- changed. Agent writes are a small part of the audit log, so the index holds only them. The
-- table is partitioned, which rules out building it concurrently.
CREATE INDEX IF NOT EXISTS "idx_audit_entries_agent_writes" ON "audit_entries"("organization_id", "business_unit_id", "timestamp" DESC) WHERE "ai_run_id" IS NOT NULL;
