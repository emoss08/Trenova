-- Plan prices are read only by move: the board, the next plan's stability and the learner
-- all look a move up. Nothing reads them by worker, so the worker index only costs writes.
DROP INDEX IF EXISTS "idx_dispatch_plan_prices_worker";
