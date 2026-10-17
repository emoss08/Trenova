ALTER TABLE "agent_analysis_runs"
    DROP COLUMN IF EXISTS "layout",
    DROP COLUMN IF EXISTS "figures",
    DROP COLUMN IF EXISTS "tables",
    DROP COLUMN IF EXISTS "title";
