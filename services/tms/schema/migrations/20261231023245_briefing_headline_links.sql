ALTER TABLE "assistant_briefings"
    ADD COLUMN IF NOT EXISTS "headline_links" JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN "assistant_briefings"."headline_links" IS 'The phrases of the headline that open the section they are about, in the order they appear: text, section key and the section''s path';
