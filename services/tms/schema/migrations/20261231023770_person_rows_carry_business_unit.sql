-- A role assignment and a person's second factor name a person (users has no organization_id),
-- so until now nothing in the database held that person to the row's tenant. Each table gets the
-- business unit of its organization, held to it by (organization_id, business_unit_id) ->
-- organizations(id, business_unit_id), and the person keys are added again carrying it, NOT VALID
-- so no table is scanned under its lock; the next migrations index, validate and swap them in under
-- the old names. Before anything changes, every row whose person (or assigner) belongs to another
-- business unit than its organization is counted, and the migration stops with each table, column
-- and count rather than writing a business unit the data contradicts.

DO $preflight$
DECLARE
    k record;
    found bigint;
    problems text := '';
BEGIN
    FOR k IN
        SELECT * FROM (VALUES
        ('user_role_assignments', 'user_id'),
        ('user_role_assignments', 'assigned_by'),
        ('mfa_authenticators', 'user_id'),
        ('mfa_recovery_codes', 'user_id')
        ) AS v(child, person)
    LOOP
        EXECUTE format(
            'SELECT count(*) FROM %I c JOIN organizations o ON o.id = c.organization_id '
            'JOIN users u ON u.id = c.%I WHERE u.business_unit_id IS DISTINCT FROM o.business_unit_id',
            k.child, k.person
        ) INTO found;
        IF found > 0 THEN
            problems := problems || format(E'\n  %s.%s: %s rows name a person of another tenant',
                k.child, k.person, found);
        END IF;
    END LOOP;
    IF problems <> '' THEN
        RAISE EXCEPTION 'rows name a person outside their organization''s tenant, so their keys cannot carry the tenant; nothing was changed:%', problems
            USING HINT = 'Delete each such row (a role assignment or second factor of a person who is not in the organization), then run the migration again (docs/engineering/backend/data-access.md, "Tenant keys").';
    END IF;
END
$preflight$;

ALTER TABLE "user_role_assignments"
    ADD COLUMN "business_unit_id" varchar(100);

ALTER TABLE "mfa_authenticators"
    ADD COLUMN "business_unit_id" varchar(100);

ALTER TABLE "mfa_recovery_codes"
    ADD COLUMN "business_unit_id" varchar(100);

UPDATE "user_role_assignments" AS c
SET "business_unit_id" = o."business_unit_id"
FROM "organizations" AS o
WHERE o."id" = c."organization_id";

UPDATE "mfa_authenticators" AS c
SET "business_unit_id" = o."business_unit_id"
FROM "organizations" AS o
WHERE o."id" = c."organization_id";

UPDATE "mfa_recovery_codes" AS c
SET "business_unit_id" = o."business_unit_id"
FROM "organizations" AS o
WHERE o."id" = c."organization_id";

ALTER TABLE "user_role_assignments"
    ALTER COLUMN "business_unit_id" SET NOT NULL,
    ADD CONSTRAINT "fk_user_role_assignments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE NOT VALID,
    ADD CONSTRAINT "tk_80defa39a224dc9fb1c525f2" FOREIGN KEY ("user_id", "business_unit_id") REFERENCES "users"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE NOT VALID,
    ADD CONSTRAINT "tk_857bef43d07b52ed5702ad7d" FOREIGN KEY ("assigned_by", "business_unit_id") REFERENCES "users"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL ("assigned_by") NOT VALID,
    ADD CONSTRAINT "tk_700e10ca42bf930a785b5549" FOREIGN KEY ("organization_id", "business_unit_id") REFERENCES "organizations"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE NOT VALID;

ALTER TABLE "mfa_authenticators"
    ALTER COLUMN "business_unit_id" SET NOT NULL,
    ADD CONSTRAINT "fk_mfa_authenticators_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE NOT VALID,
    ADD CONSTRAINT "tk_fd4147c0d23417a221516c3a" FOREIGN KEY ("user_id", "business_unit_id") REFERENCES "users"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE NOT VALID,
    ADD CONSTRAINT "tk_e4891f3a11f5859fe64dcf52" FOREIGN KEY ("organization_id", "business_unit_id") REFERENCES "organizations"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE NOT VALID;

ALTER TABLE "mfa_recovery_codes"
    ALTER COLUMN "business_unit_id" SET NOT NULL,
    ADD CONSTRAINT "fk_mfa_recovery_codes_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE NOT VALID,
    ADD CONSTRAINT "tk_f0bf252abab7056ce9aa5215" FOREIGN KEY ("user_id", "business_unit_id") REFERENCES "users"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE NOT VALID,
    ADD CONSTRAINT "tk_aed69198d06e38b0729d9648" FOREIGN KEY ("organization_id", "business_unit_id") REFERENCES "organizations"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE NOT VALID;
