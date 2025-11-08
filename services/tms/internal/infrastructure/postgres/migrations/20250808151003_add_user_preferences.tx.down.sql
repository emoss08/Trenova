DROP TRIGGER IF EXISTS user_preferences_update_timestamp_trigger ON user_preferences;

--bun:split
DROP FUNCTION IF EXISTS user_preferences_update_timestamp();

--bun:split
DROP INDEX IF EXISTS "idx_user_preferences_preferences_gin";

DROP INDEX IF EXISTS "idx_user_preferences_created_at";

DROP INDEX IF EXISTS "idx_user_preferences_organization";

DROP INDEX IF EXISTS "idx_user_preferences_user";

--bun:split
DROP TABLE IF EXISTS "user_preferences";

