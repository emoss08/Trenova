--keel:lint_ignore table-rewrite
-- Every communication profile has its own inbound AS2 address, generated with the row so no writer
-- can miss it. The volatile default gives each existing row its own value; the table holds a few
-- rows per trading partner, so the rewrite is short.
ALTER TABLE "edi_communication_profiles"
    ADD COLUMN IF NOT EXISTS "as2_inbound_token" varchar(64) NOT NULL
    DEFAULT (replace(gen_random_uuid()::text, '-', '') || replace(gen_random_uuid()::text, '-', ''));

COMMENT ON COLUMN "edi_communication_profiles"."as2_inbound_token" IS 'The unguessable part of the URL a partner posts AS2 messages to; it names this profile and so its tenant';
