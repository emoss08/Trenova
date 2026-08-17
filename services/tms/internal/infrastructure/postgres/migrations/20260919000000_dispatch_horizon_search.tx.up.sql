ALTER TABLE "dispatch_controls"
    ADD COLUMN IF NOT EXISTS "horizon_search_iterations" SMALLINT;

--bun:split
COMMENT ON COLUMN "dispatch_controls"."horizon_search_iterations" IS 'Local-search passes used to refine a horizon plan after the initial greedy pass; NULL uses the default, 0 disables the search and keeps the greedy plan, ignored when planning_mode is Immediate';
