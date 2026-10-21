ALTER TABLE "ai_work_units"
    DROP COLUMN IF EXISTS "advisor_cost_usd",
    DROP COLUMN IF EXISTS "advisor_output_tokens",
    DROP COLUMN IF EXISTS "advisor_prompt_tokens",
    DROP COLUMN IF EXISTS "advisor_calls";
