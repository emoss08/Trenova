-- The sender's own words in an email: everything before a quoted reply, a
-- forwarded header or a signature begins, without quoted lines. It mirrors
-- stringutils.MailBody, which the indexer embeds, so the keyword and the
-- meaning search of the inbox read the same text. A quoted thread would make
-- every message in it match every search for anything said in it.
CREATE OR REPLACE FUNCTION mail_own_words(body text)
RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE
AS $$
    SELECT btrim(regexp_replace(
        CASE WHEN parts.boundary > 0 THEN left(parts.source, parts.boundary - 1) ELSE parts.source END,
        '^[ \t]*>[^\n]*$',
        '',
        'gn'
    ), E' \t\r\n')
    FROM (
        SELECT
            COALESCE(body, '') AS source,
            regexp_instr(
                COALESCE(body, ''),
                '^(-- ?\r?$|[ \t]*(-----Original Message-----|---------- Forwarded message|________________________________|On [^\n]{4,200} wrote:[ \t\r]*$))',
                1,
                1,
                0,
                'n'
            ) AS boundary
    ) AS parts
$$;

--bun:split
-- Keyword search over the inbox by relevance: the subject weighs most, then
-- who sent it, then what they wrote. The list view keeps its own newest-first
-- filter; this is what search_inbound_messages ranks by.
ALTER TABLE "inbound_messages"
    ADD COLUMN IF NOT EXISTS "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('english', COALESCE("subject", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("from_name", '') || ' ' || COALESCE("from_address", '')), 'B') ||
        setweight(immutable_to_tsvector('english', mail_own_words("text_body")), 'C')
    ) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_inbound_messages_search_vector"
    ON "inbound_messages" USING GIN ("search_vector");

COMMENT ON COLUMN "inbound_messages"."search_vector" IS 'Subject (A), sender (B) and the sender''s own words with quoted history and signature stripped (C), for relevance search';
