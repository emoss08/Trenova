ALTER TABLE "document_controls"
    DROP CONSTRAINT IF EXISTS "ck_document_controls_capture_retention_days";

--bun:split
ALTER TABLE "document_controls"
    DROP COLUMN IF EXISTS "capture_allow_auto_update",
    DROP COLUMN IF EXISTS "capture_min_agent_version",
    DROP COLUMN IF EXISTS "capture_retention_days",
    DROP COLUMN IF EXISTS "capture_auto_file_cover_sheets",
    DROP COLUMN IF EXISTS "enable_capture";

--bun:split
DROP TABLE IF EXISTS "capture_cover_sheets";

--bun:split
DROP TABLE IF EXISTS "capture_items";

--bun:split
DROP TABLE IF EXISTS "capture_pages";

--bun:split
DROP TABLE IF EXISTS "capture_batches";

--bun:split
DROP TABLE IF EXISTS "capture_requests";

--bun:split
DROP TABLE IF EXISTS "capture_profiles";

--bun:split
DROP TABLE IF EXISTS "capture_pairings";

--bun:split
DROP TABLE IF EXISTS "capture_devices";
