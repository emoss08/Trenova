CREATE TABLE IF NOT EXISTS "user_preferences"(
    "id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "preferences" jsonb NOT NULL DEFAULT '{"dismissedNotices": [], "dismissedDialogs": [], "uiSettings": {}}' ::jsonb,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_user_preferences" PRIMARY KEY ("id"),
    CONSTRAINT "uq_user_preferences_user" UNIQUE ("user_id"),
    CONSTRAINT "fk_user_preferences_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_preferences_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_preferences_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_user_preferences_user" ON "user_preferences"("user_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_user_preferences_organization" ON "user_preferences"("organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_user_preferences_created_at" ON "user_preferences"("created_at", "updated_at");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_user_preferences_preferences_gin" ON "user_preferences" USING GIN("preferences");

--bun:split
COMMENT ON TABLE user_preferences IS 'Stores user-specific UI preferences, dismissed notices, and dialog settings';

--bun:split
COMMENT ON COLUMN user_preferences.preferences IS 'JSONB structure containing dismissedNotices, dismissedDialogs, and uiSettings';

--bun:split
CREATE OR REPLACE FUNCTION user_preferences_update_timestamp()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS user_preferences_update_timestamp_trigger ON user_preferences;

CREATE TRIGGER user_preferences_update_timestamp_trigger
    BEFORE UPDATE ON user_preferences
    FOR EACH ROW
    EXECUTE FUNCTION user_preferences_update_timestamp();

--bun:split
ALTER TABLE user_preferences
    ALTER COLUMN user_id SET STATISTICS 1000;

ALTER TABLE user_preferences
    ALTER COLUMN organization_id SET STATISTICS 1000;

