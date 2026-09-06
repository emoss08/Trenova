DROP INDEX IF EXISTS "idx_worker_profiles_clearinghouse_due";
--bun:split
DROP INDEX IF EXISTS "idx_worker_profiles_drug_alcohol";
--bun:split
ALTER TABLE "worker_profiles"
    DROP COLUMN IF EXISTS "next_clearinghouse_query_due",
    DROP COLUMN IF EXISTS "last_clearinghouse_query_at",
    DROP COLUMN IF EXISTS "return_to_duty_status",
    DROP COLUMN IF EXISTS "drug_alcohol_status";
--bun:split
DROP TABLE IF EXISTS "worker_clearinghouse_queries";
--bun:split
DROP TABLE IF EXISTS "worker_dot_violations";
--bun:split
DROP TABLE IF EXISTS "worker_dot_tests";
--bun:split
DROP TABLE IF EXISTS "dot_random_draw_entries";
--bun:split
DROP TABLE IF EXISTS "dot_random_draws";
--bun:split
DROP TABLE IF EXISTS "dot_random_pools";
--bun:split
DROP TYPE IF EXISTS "worker_return_to_duty_status_enum";
--bun:split
DROP TYPE IF EXISTS "worker_drug_alcohol_status_enum";
--bun:split
DROP TYPE IF EXISTS "dot_clearinghouse_result_enum";
--bun:split
DROP TYPE IF EXISTS "dot_clearinghouse_query_type_enum";
--bun:split
DROP TYPE IF EXISTS "dot_violation_status_enum";
--bun:split
DROP TYPE IF EXISTS "dot_violation_type_enum";
--bun:split
DROP TYPE IF EXISTS "dot_random_entry_status_enum";
--bun:split
DROP TYPE IF EXISTS "dot_random_draw_status_enum";
--bun:split
DROP TYPE IF EXISTS "dot_random_period_enum";
--bun:split
DROP TYPE IF EXISTS "dot_test_result_enum";
--bun:split
DROP TYPE IF EXISTS "dot_test_status_enum";
--bun:split
DROP TYPE IF EXISTS "dot_test_substance_enum";
--bun:split
DROP TYPE IF EXISTS "dot_test_type_enum";
