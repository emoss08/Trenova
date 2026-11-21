-- A user's current organization was held to the user's business unit by a trigger alone, and
-- its foreign key named the organization by id with ON DELETE CASCADE, so a row the trigger
-- never saw would have let removing one tenant's organization delete another tenant's user. The
-- reference now carries the user's business unit, and an organization still somebody's current
-- one cannot be deleted: a removal moves or deletes the tenant's people before its organization.

ALTER TABLE "users"
    DROP CONSTRAINT IF EXISTS "fk_users_current_organization";

ALTER TABLE "users"
    ADD CONSTRAINT "fk_users_current_organization" FOREIGN KEY ("current_organization_id", "business_unit_id") REFERENCES "organizations"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE NO ACTION;

DROP TRIGGER IF EXISTS "trigger_check_user_organization_business_unit" ON "users";

DROP FUNCTION IF EXISTS "check_user_organization_business_unit"();
