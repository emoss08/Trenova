-- A conversation is named three ways: from its first question as it opens,
-- by the model once that question has been read, and by the person. The
-- model's name replaces only a name taken from the question, never one a
-- person gave, so the thread says where its title came from.
ALTER TABLE "assistant_threads"
    ADD COLUMN IF NOT EXISTS "title_source" varchar(20) NOT NULL DEFAULT 'Question';

ALTER TABLE "assistant_threads"
    DROP CONSTRAINT IF EXISTS "ck_assistant_threads_title_source";

ALTER TABLE "assistant_threads"
    ADD CONSTRAINT "ck_assistant_threads_title_source" CHECK ("title_source" IN ('Question', 'Model', 'Person'));

COMMENT ON COLUMN "assistant_threads"."title_source" IS 'Where the conversation''s title came from: its first question, the model that named it, or the person';
