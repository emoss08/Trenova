ALTER TABLE IF EXISTS "dash_controls"
    DROP COLUMN IF EXISTS "require_contact_change_approval";

--bun:split
DROP TABLE IF EXISTS "worker_profile_change_requests";

--bun:split
DROP TABLE IF EXISTS "worker_policy_acknowledgements";

--bun:split
DROP TABLE IF EXISTS "worker_policies";

--bun:split
DROP TYPE IF EXISTS "profile_change_status_enum";

--bun:split
DROP TYPE IF EXISTS "worker_policy_audience_enum";
