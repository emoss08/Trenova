-- Each provider now carries its own limits: how long one call may take, how
-- many calls it takes at once across every server, and the most it may spend
-- in a calendar month with what happens to a task once it has. The key gains
-- what can be said about it without the secret, and a rotation keeps the key
-- it replaced for a day so calls refused with the new one still go through.
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "timeout_seconds" integer NOT NULL DEFAULT 60,
    ADD COLUMN IF NOT EXISTS "max_concurrent" integer NOT NULL DEFAULT 8,
    ADD COLUMN IF NOT EXISTS "monthly_cap_usd" numeric(12, 2),
    ADD COLUMN IF NOT EXISTS "on_cap" varchar(10) NOT NULL DEFAULT 'Next',
    ADD COLUMN IF NOT EXISTS "api_key_prefix" varchar(20),
    ADD COLUMN IF NOT EXISTS "api_key_last_four" varchar(4),
    ADD COLUMN IF NOT EXISTS "api_key_added_at" bigint,
    ADD COLUMN IF NOT EXISTS "api_key_added_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "api_key_last_used_at" bigint,
    ADD COLUMN IF NOT EXISTS "previous_api_key" text,
    ADD COLUMN IF NOT EXISTS "rotation_expires_at" bigint;

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_timeout_seconds" CHECK ("timeout_seconds" BETWEEN 5 AND 600);

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_max_concurrent" CHECK ("max_concurrent" BETWEEN 1 AND 64);

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_monthly_cap_usd" CHECK ("monthly_cap_usd" IS NULL OR "monthly_cap_usd" > 0);

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_on_cap" CHECK ("on_cap" IN ('Next', 'Stop'));

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_api_key_last_four" CHECK ("api_key_last_four" IS NULL OR char_length("api_key_last_four") = 4);

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_previous_api_key" CHECK (("previous_api_key" IS NULL) = ("rotation_expires_at" IS NULL));

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "fk_ai_providers_api_key_added_by" FOREIGN KEY ("api_key_added_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
-- A key stored before now has no record of when it was added; the provider's
-- creation is the closest thing known. Its prefix and last four are read
-- from the decrypted key the first time the application loads it.
UPDATE "ai_providers"
SET "api_key_added_at" = "created_at"
WHERE "api_key" IS NOT NULL
    AND "api_key" <> ''
    AND "api_key_added_at" IS NULL;

--bun:split
COMMENT ON COLUMN "ai_providers"."timeout_seconds" IS 'How long one call may take before it counts as this provider failing and the task moves on';

--bun:split
COMMENT ON COLUMN "ai_providers"."max_concurrent" IS 'Calls this provider takes at once across every server; past it work moves to the next provider';

--bun:split
COMMENT ON COLUMN "ai_providers"."monthly_cap_usd" IS 'Most this provider may spend in a UTC calendar month, over priced calls; null means no cap';

--bun:split
COMMENT ON COLUMN "ai_providers"."on_cap" IS 'What happens to a task once the monthly cap is reached: Next moves it on, Stop fails it';

--bun:split
COMMENT ON COLUMN "ai_providers"."api_key_prefix" IS 'Vendor prefix of the stored key, kept in the clear to tell keys apart';

--bun:split
COMMENT ON COLUMN "ai_providers"."api_key_last_four" IS 'Last four characters of the stored key, kept in the clear to tell keys apart';

--bun:split
COMMENT ON COLUMN "ai_providers"."api_key_added_at" IS 'When the stored key was entered';

--bun:split
COMMENT ON COLUMN "ai_providers"."api_key_added_by_id" IS 'Who entered the stored key';

--bun:split
COMMENT ON COLUMN "ai_providers"."api_key_last_used_at" IS 'Last time a call succeeded with the stored key, written at most once a minute';

--bun:split
COMMENT ON COLUMN "ai_providers"."previous_api_key" IS 'Key a rotation replaced, encrypted like api_key, used only when the new key is refused';

--bun:split
COMMENT ON COLUMN "ai_providers"."rotation_expires_at" IS 'When the replaced key stops being used as a fallback';
