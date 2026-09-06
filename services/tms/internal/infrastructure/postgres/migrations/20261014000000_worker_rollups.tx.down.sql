DROP INDEX IF EXISTS "idx_worker_profiles_next_credential_expiry";

--bun:split

DROP INDEX IF EXISTS "idx_worker_profiles_rollups";

--bun:split

ALTER TABLE "worker_profiles"
    DROP COLUMN IF EXISTS "next_training_due",
    DROP COLUMN IF EXISTS "next_credential_expiry",
    DROP COLUMN IF EXISTS "safety_score",
    DROP COLUMN IF EXISTS "safety_rating",
    DROP COLUMN IF EXISTS "training_health";

--bun:split

DROP TYPE IF EXISTS "worker_safety_rating_enum";

--bun:split

DROP TYPE IF EXISTS "worker_training_health_enum";
