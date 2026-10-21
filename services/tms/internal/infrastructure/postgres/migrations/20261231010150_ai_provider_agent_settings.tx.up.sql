-- An administrator turns on, per chat provider, what the desk bench used to
-- be the only way to reach: Anthropic's advisor on every chat call routed to
-- the provider, and code mode (read-only scripts, and scripts that propose
-- writes as one batch) for the provider's model. Every one is off by default.
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "advisor_enabled" boolean NOT NULL DEFAULT FALSE;

--bun:split
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "advisor_model" varchar(200);

--bun:split
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "code_mode_scripts" boolean NOT NULL DEFAULT FALSE;

--bun:split
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "code_mode_writes" boolean NOT NULL DEFAULT FALSE;

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_advisor_kind" CHECK (NOT "advisor_enabled" OR "kind" = 'AnthropicMessages');

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_code_mode_writes" CHECK (NOT "code_mode_writes" OR "code_mode_scripts");

--bun:split
COMMENT ON COLUMN "ai_providers"."advisor_enabled" IS 'Every chat call routed to this provider declares Anthropic''s advisor tool';

--bun:split
COMMENT ON COLUMN "ai_providers"."advisor_model" IS 'The model the advisor runs on; null is the default advisor (claude-opus-5-5)';

--bun:split
COMMENT ON COLUMN "ai_providers"."code_mode_scripts" IS 'The provider''s model may answer with a read-only script (code mode)';

--bun:split
COMMENT ON COLUMN "ai_providers"."code_mode_writes" IS 'The provider''s scripts may prepare writes, proposed as one batch for a person; needs code_mode_scripts';
