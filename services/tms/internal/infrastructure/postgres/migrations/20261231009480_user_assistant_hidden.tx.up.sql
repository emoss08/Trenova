-- Whether the person has turned the assistant off. When it is, the shell does not mount the
-- assistant at all: no beacon, no edge tab, no panel and no shortcut, on every page and every
-- device they sign in from. Stored as "hidden" so every existing row, and every row a later
-- insert writes without naming it, keeps the assistant where it is today.
ALTER TABLE "users"
    ADD COLUMN IF NOT EXISTS "assistant_hidden" boolean NOT NULL DEFAULT false;
