-- A command palette question is answered on the Nova harness and saved
-- nowhere until the person opens it in the Desk, which starts an ordinary
-- Desk conversation. The hidden conversations unkept quick questions were
-- answered on go, and so does their origin.
DELETE FROM "assistant_threads"
WHERE "origin" = 'Ask';

ALTER TABLE "assistant_threads"
    DROP CONSTRAINT IF EXISTS "ck_assistant_threads_origin";

ALTER TABLE "assistant_threads"
    ADD CONSTRAINT "ck_assistant_threads_origin" CHECK (
        "origin" IN ('Panel', 'Desk', 'Watchtower', 'Briefing', 'Import', 'Formula')
    );
