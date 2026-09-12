--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

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
