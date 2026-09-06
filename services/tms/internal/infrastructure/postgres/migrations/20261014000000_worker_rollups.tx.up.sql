-- Roster roll-ups.
--
-- Training health and the safety rating are computed by scanning a worker's
-- whole history, which is fine for one worker on a panel and impossible for a
-- page of fifty on the roster. These columns are the same answer, denormalised
-- so the list can filter and sort on it, maintained the way compliance_status
-- already is: written when the underlying record changes and re-checked by the
-- nightly compliance sweep, which is what catches the time-based transitions
-- nothing writes (a course becoming overdue at midnight, points rolling off).
--
-- The ledger of record is still worker_training_records and
-- worker_safety_events. These columns are a cache and may be rebuilt from them
-- at any time.

CREATE TYPE "worker_training_health_enum" AS ENUM(
    'Current',
    'Scheduled',
    'DueSoon',
    'ExpiringSoon',
    'Overdue',
    'Expired',
    'Failed',
    'Missing'
);

--bun:split

CREATE TYPE "worker_safety_rating_enum" AS ENUM(
    'Excellent',
    'Good',
    'Watch',
    'AtRisk'
);

--bun:split

ALTER TABLE "worker_profiles"
    ADD COLUMN IF NOT EXISTS "training_health" worker_training_health_enum NOT NULL DEFAULT 'Current',
    ADD COLUMN IF NOT EXISTS "safety_rating" worker_safety_rating_enum NOT NULL DEFAULT 'Excellent',
    ADD COLUMN IF NOT EXISTS "safety_score" smallint NOT NULL DEFAULT 100,
    ADD COLUMN IF NOT EXISTS "next_credential_expiry" bigint,
    ADD COLUMN IF NOT EXISTS "next_training_due" bigint;

--bun:split

COMMENT ON COLUMN "worker_profiles"."training_health" IS
    'Worst health across the worker''s required courses. Cache of BuildTrainingSummary.';

--bun:split

COMMENT ON COLUMN "worker_profiles"."safety_rating" IS
    'Cache of BuildSafetyScorecard.Rating. worker_safety_events remains the ledger of record.';

--bun:split

COMMENT ON COLUMN "worker_profiles"."next_credential_expiry" IS
    'Earliest expiry among required credentials, for the roster''s expiring-soon view. Null when nothing expires.';

--bun:split

-- The roster filters on these together (show me everyone who is not clean), so
-- one composite index serves the common case rather than three separate scans.
CREATE INDEX IF NOT EXISTS "idx_worker_profiles_rollups"
    ON "worker_profiles" ("organization_id", "business_unit_id", "compliance_status", "training_health", "safety_rating");

--bun:split

-- Sorting the roster by what expires next, and the credential forecast, both
-- read this ordering. Partial because a worker with nothing expiring never
-- appears in that view.
CREATE INDEX IF NOT EXISTS "idx_worker_profiles_next_credential_expiry"
    ON "worker_profiles" ("organization_id", "business_unit_id", "next_credential_expiry")
    WHERE "next_credential_expiry" IS NOT NULL;
