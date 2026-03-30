ALTER TABLE "organizations"
    ADD COLUMN IF NOT EXISTS "login_slug" varchar(100);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "idx_organizations_login_slug" ON "organizations"("login_slug")
WHERE
    "login_slug" IS NOT NULL
    AND "login_slug" <> '';
