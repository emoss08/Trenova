-- Horizon planning now anneals over the board for a number of rounds, each a pass over
-- every move, rather than counting single search iterations.
ALTER TABLE "dispatch_controls" RENAME COLUMN "horizon_search_iterations" TO "horizon_search_rounds";
