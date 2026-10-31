CREATE TABLE IF NOT EXISTS "assistant_palette_questions" (
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "user_id" VARCHAR(100) NOT NULL,
    "status" VARCHAR(20) NOT NULL,
    "kept_thread_id" VARCHAR(100),
    "keep_claimed_at" BIGINT,
    "finished_at" BIGINT,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    "updated_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_assistant_palette_questions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_assistant_palette_questions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_palette_questions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_palette_questions_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_assistant_palette_questions_status" CHECK ("status" IN ('Asking', 'Answered', 'Failed', 'Stopped'))
);

-- The allowance counts a person's questions this month, and the palette counts
-- the ones still being answered.
CREATE INDEX IF NOT EXISTS "idx_assistant_palette_questions_user_created" ON "assistant_palette_questions" ("organization_id", "business_unit_id", "user_id", "created_at" DESC);

-- The retention sweep removes what has aged out across every tenant.
CREATE INDEX IF NOT EXISTS "idx_assistant_palette_questions_created" ON "assistant_palette_questions" ("created_at");

COMMENT ON TABLE "assistant_palette_questions" IS 'Questions asked from the command palette: answered on the Nova harness, counted against the person''s AI allowance, and carried into a Desk conversation at most once';

COMMENT ON COLUMN "assistant_palette_questions"."kept_thread_id" IS 'The Desk conversation the question was carried into; once set, the question counts as that conversation''s message instead of on its own';

COMMENT ON COLUMN "assistant_palette_questions"."keep_claimed_at" IS 'When carrying the question into the Desk began, so a second request does not open a second conversation';
