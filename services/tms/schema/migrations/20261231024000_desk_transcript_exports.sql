-- Requests to take Desk conversations out of Trenova as a zip of transcripts, and the file each
-- produced. thread_ids fixes the conversations a file holds when it is asked for, so the count the
-- requester was shown and the file agree however the conversations change while it is written. One
-- export is written at a time per organization: the partial unique index refuses a second while one
-- is pending or running, which holds under concurrent requests where a check in the service would not.
CREATE TABLE IF NOT EXISTS "desk_transcript_exports"(
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "requested_by_user_id" VARCHAR(100) NOT NULL,
    "kind" VARCHAR(20) NOT NULL,
    "filter" JSONB,
    "thread_ids" JSONB NOT NULL DEFAULT '[]'::jsonb,
    "coverage" JSONB NOT NULL DEFAULT '{}'::jsonb,
    "conversation_count" BIGINT NOT NULL DEFAULT 0,
    "status" VARCHAR(20) NOT NULL DEFAULT 'Pending',
    "byte_size" BIGINT NOT NULL DEFAULT 0,
    "sha256" VARCHAR(64),
    "artifact_key" VARCHAR(500),
    "artifact_expires_at" BIGINT,
    "workflow_id" VARCHAR(255),
    "error_message" TEXT,
    "started_at" BIGINT,
    "completed_at" BIGINT,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_desk_transcript_exports" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "ck_desk_transcript_exports_kind" CHECK ("kind" IN ('Filter', 'Selection')),
    CONSTRAINT "ck_desk_transcript_exports_status" CHECK ("status" IN ('Pending', 'Running', 'Succeeded', 'Failed', 'Expired')),
    CONSTRAINT "ck_desk_transcript_exports_count" CHECK ("conversation_count" BETWEEN 0 AND 5000),
    CONSTRAINT "fk_desk_transcript_exports_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_desk_transcript_exports_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

COMMENT ON TABLE "desk_transcript_exports" IS 'Requests to export other people''s Desk conversations as a zip of transcripts, and the file each produced';

COMMENT ON COLUMN "desk_transcript_exports"."thread_ids" IS 'The conversations the file holds, fixed when the export was requested';

COMMENT ON COLUMN "desk_transcript_exports"."coverage" IS 'What the export covers in words and dates: the timeframe, the person and agent named, the owners included';

CREATE INDEX IF NOT EXISTS "idx_desk_transcript_exports_listing" ON "desk_transcript_exports"("organization_id", "business_unit_id", "created_at" DESC, "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_desk_transcript_exports_expiry" ON "desk_transcript_exports"("artifact_expires_at") WHERE "artifact_key" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_desk_transcript_exports_live" ON "desk_transcript_exports"("updated_at") WHERE "status" IN ('Pending', 'Running');

CREATE UNIQUE INDEX IF NOT EXISTS "uq_desk_transcript_exports_one_live" ON "desk_transcript_exports"("organization_id") WHERE "status" IN ('Pending', 'Running');
