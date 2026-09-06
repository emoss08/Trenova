DELETE FROM "worker_credential_types"
WHERE "is_system"
    AND "code" IN ('ROAD_TEST', 'ANNUAL_REVIEW', 'VIOLATION_CERT');
--bun:split
ALTER TABLE "data_retention"
    DROP COLUMN IF EXISTS "driver_qualification_retention_period";
--bun:split
DROP TABLE IF EXISTS "worker_employment_verifications";
--bun:split
DROP TYPE IF EXISTS "employment_verification_method_enum";
--bun:split
DROP TYPE IF EXISTS "employment_verification_status_enum";
