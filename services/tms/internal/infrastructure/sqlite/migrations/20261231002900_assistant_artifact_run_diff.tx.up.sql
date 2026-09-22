-- Hand-written. The dialect converter never emits a CHECK alteration, so the
-- Postgres migration of this name — which added 'run_diff' to the kind list —
-- has no generated counterpart, and SQLite has been rejecting every artifact
-- compare_report_runs produces.
--
-- SQLite cannot alter a CHECK in place, so the table is rebuilt. It is rebuilt
-- WITHOUT the kind CHECK rather than with a longer one: the converter cannot
-- maintain it, so any future kind would break here again the same silent way.
-- assistantartifact.Kind.IsValid is the gate that is actually enforced on
-- every write; Postgres keeps the CHECK as a second line, and SQLite joins the
-- other constraints this dialect already goes without.
CREATE TABLE "assistant_artifacts_rebuilt"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "thread_id" TEXT NOT NULL,
    "message_id" TEXT,
    "run_id" TEXT,
    "proposal_id" TEXT,
    "plan_id" TEXT,
    "kind" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Ready',
    "title" TEXT NOT NULL,
    "payload" TEXT NOT NULL DEFAULT '{}',
    "source_tool_call_id" TEXT NOT NULL DEFAULT '',
    "pinned" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_assistant_artifacts" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_assistant_artifacts_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_artifacts_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_artifacts_thread" FOREIGN KEY ("thread_id", "business_unit_id", "organization_id") REFERENCES "assistant_threads"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_assistant_artifacts_status" CHECK ("status" IN ('Pending', 'Ready', 'Failed', 'Sent'))
);

--bun:split

INSERT INTO "assistant_artifacts_rebuilt" SELECT * FROM "assistant_artifacts";

--bun:split

DROP TABLE "assistant_artifacts";

--bun:split

ALTER TABLE "assistant_artifacts_rebuilt" RENAME TO "assistant_artifacts";

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_assistant_artifacts_source" ON "assistant_artifacts" ("thread_id", "source_tool_call_id", "kind")WHERE "source_tool_call_id" <> '';

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_assistant_artifacts_proposal" ON "assistant_artifacts" ("thread_id", "proposal_id")WHERE "proposal_id" IS NOT NULL;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_assistant_artifacts_plan" ON "assistant_artifacts" ("thread_id", "plan_id")WHERE "plan_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_assistant_artifacts_thread" ON "assistant_artifacts" ("organization_id", "business_unit_id", "thread_id", "pinned", "created_at" DESC);
