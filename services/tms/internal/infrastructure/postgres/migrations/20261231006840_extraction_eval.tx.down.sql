DROP INDEX IF EXISTS "idx_extraction_eval_results_case";

--bun:split
DROP TABLE IF EXISTS "extraction_eval_results";

--bun:split
DROP INDEX IF EXISTS "uq_extraction_eval_runs_active";

--bun:split
DROP INDEX IF EXISTS "idx_extraction_eval_runs_created";

--bun:split
DROP TABLE IF EXISTS "extraction_eval_runs";

--bun:split
DROP INDEX IF EXISTS "uq_extraction_eval_cases_source_correction";

--bun:split
DROP INDEX IF EXISTS "idx_extraction_eval_cases_status";

--bun:split
DROP TABLE IF EXISTS "extraction_eval_cases";
