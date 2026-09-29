-- One row per scheduled or requested retraining of the document extraction
-- model: the decision, the training export it started, and the trained
-- model's score against production on the export's validation set.
CREATE TABLE IF NOT EXISTS "ai_retraining_cycles" (
    "id" varchar(100) NOT NULL,
    "task" varchar(50) NOT NULL,
    "trigger" varchar(20) NOT NULL,
    "status" varchar(20) NOT NULL,
    "skip_reason" varchar(30),
    "requested_by" varchar(255) NOT NULL,
    "note" text,
    "export_id" varchar(100),
    "captured_from" bigint NOT NULL,
    "captured_to" bigint NOT NULL,
    "new_since" bigint NOT NULL,
    "new_examples" integer NOT NULL DEFAULT 0,
    "min_new_examples" integer NOT NULL DEFAULT 0,
    "drifting_providers" integer NOT NULL DEFAULT 0,
    "max_per_organization" integer NOT NULL,
    "validation_percent" integer NOT NULL,
    "structured_output_mode" varchar(20) NOT NULL,
    "min_accuracy_percent" integer NOT NULL,
    "max_regression_points" integer NOT NULL,
    "trainer" varchar(255),
    "attempts" integer NOT NULL DEFAULT 0,
    "claimed_at" bigint,
    "lease_expires_at" bigint,
    "training_config" varchar(1024),
    "run_directory" varchar(1024),
    "model_directory" varchar(1024),
    "prompt_sha256" varchar(64),
    "examples" integer NOT NULL DEFAULT 0,
    "model_correct" integer NOT NULL DEFAULT 0,
    "model_scored" integer NOT NULL DEFAULT 0,
    "baseline_correct" integer NOT NULL DEFAULT 0,
    "baseline_scored" integer NOT NULL DEFAULT 0,
    "gate_message" text,
    "failure_message" text,
    "finished_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_ai_retraining_cycles" PRIMARY KEY ("id"),
    CONSTRAINT "ck_ai_retraining_cycles_task" CHECK ("task" IN ('ShipmentDraftExtraction')),
    CONSTRAINT "ck_ai_retraining_cycles_trigger" CHECK ("trigger" IN ('Scheduled', 'Drift', 'Manual')),
    CONSTRAINT "ck_ai_retraining_cycles_status" CHECK ("status" IN ('Skipped', 'Exporting', 'Ready', 'Training', 'Passed', 'Rejected', 'Failed', 'Canceled')),
    CONSTRAINT "ck_ai_retraining_cycles_skip_reason" CHECK ("skip_reason" IS NULL OR "skip_reason" IN ('CycleOpen', 'ExportActive', 'TooSoon', 'NotEnoughExamples')),
    CONSTRAINT "ck_ai_retraining_cycles_skipped" CHECK (("status" = 'Skipped') = ("skip_reason" IS NOT NULL)),
    CONSTRAINT "ck_ai_retraining_cycles_structured_output_mode" CHECK ("structured_output_mode" IN ('JSONSchema', 'JSONMode', 'Prompted')),
    CONSTRAINT "ck_ai_retraining_cycles_window" CHECK ("captured_from" >= 0 AND "captured_from" < "captured_to" AND "new_since" BETWEEN "captured_from" AND "captured_to"),
    CONSTRAINT "ck_ai_retraining_cycles_max_per_organization" CHECK ("max_per_organization" BETWEEN 1 AND 20000),
    CONSTRAINT "ck_ai_retraining_cycles_validation_percent" CHECK ("validation_percent" BETWEEN 1 AND 50),
    CONSTRAINT "ck_ai_retraining_cycles_gate" CHECK ("min_accuracy_percent" BETWEEN 0 AND 100 AND "max_regression_points" BETWEEN 0 AND 100),
    CONSTRAINT "ck_ai_retraining_cycles_lease" CHECK ("status" <> 'Training' OR ("trainer" IS NOT NULL AND "lease_expires_at" IS NOT NULL)),
    CONSTRAINT "fk_ai_retraining_cycles_export" FOREIGN KEY ("export_id") REFERENCES "ai_training_exports"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ai_retraining_cycles_created"
    ON "ai_retraining_cycles" ("created_at" DESC);

--bun:split
-- At most one cycle is exporting, waiting for a trainer, or training.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_ai_retraining_cycles_open"
    ON "ai_retraining_cycles" ("task")
    WHERE "status" IN ('Exporting', 'Ready', 'Training');
