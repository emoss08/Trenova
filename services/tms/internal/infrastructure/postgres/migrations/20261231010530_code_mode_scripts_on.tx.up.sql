ALTER TABLE "ai_providers"
    ALTER COLUMN "code_mode_scripts" SET DEFAULT TRUE;

--bun:split
UPDATE "ai_providers"
SET "code_mode_scripts" = TRUE
WHERE 'AssistantChat' = ANY ("tasks");

--bun:split
COMMENT ON COLUMN "ai_providers"."code_mode_scripts" IS 'The provider''s model may answer with a read-only script (code mode), run in Trenova''s own interpreter; on by default';
