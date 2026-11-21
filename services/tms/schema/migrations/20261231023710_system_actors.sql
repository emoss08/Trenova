-- Background work that acts for nobody in particular (automatic transfers, period close, EDI
-- comments, unattended agents) is written under a system actor. The actor was found by its
-- username, server-wide, so on a shared server the first person anyone named "system" would
-- have become the actor of every tenant, and other tenants' rows would have referred to that
-- person. Each business unit now has exactly one system actor, named by this table rather than
-- by any value a person can choose, and it never signs in.

CREATE TABLE IF NOT EXISTS "system_actors"(
    "business_unit_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    CONSTRAINT "pk_system_actors" PRIMARY KEY ("business_unit_id"),
    CONSTRAINT "uq_system_actors_user" UNIQUE ("user_id"),
    CONSTRAINT "fk_system_actors_user" FOREIGN KEY ("user_id", "business_unit_id") REFERENCES "users"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_system_actors_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

COMMENT ON TABLE "system_actors" IS 'The one users row of each business unit that background work is attributed to; it never signs in';

SELECT trenova_rls.apply_policy(
    'public.system_actors',
    $expr$business_unit_id = (SELECT trenova_rls.bu_id())$expr$
);

INSERT INTO trenova_rls.append_only_tables("table_name", "reason")
VALUES ('system_actors', 'A business unit''s system actor is chosen once, when the tenant is made; only a removal, as a bypass member, deletes it')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";

INSERT INTO "system_actors"("business_unit_id", "user_id")
SELECT u."business_unit_id", u."id"
FROM "users" u
WHERE lower(u."username") = 'system'
    AND lower(u."email_address") = 'system@trenova.app'
ON CONFLICT DO NOTHING;

INSERT INTO "users"(
    "id", "business_unit_id", "current_organization_id", "status", "name", "username",
    "password", "email_address", "timezone"
)
SELECT
    'usr_' || substr(substr(b."id", 4), 1, 10) || upper(substr(md5('system-actor:' || b."id"), 1, 16)),
    b."id",
    first_org."id",
    'Active',
    'System Account',
    'system-' || lower(right(b."id", 13)),
    '!',
    'system-' || lower(right(b."id", 13)) || '@system.trenova.invalid',
    'UTC'
FROM "business_units" b
CROSS JOIN LATERAL (
    SELECT o."id"
    FROM "organizations" o
    WHERE o."business_unit_id" = b."id"
    ORDER BY o."created_at", o."id"
    LIMIT 1
) first_org
WHERE NOT EXISTS (SELECT 1 FROM "system_actors" sa WHERE sa."business_unit_id" = b."id");

INSERT INTO "system_actors"("business_unit_id", "user_id")
SELECT b."id", 'usr_' || substr(substr(b."id", 4), 1, 10) || upper(substr(md5('system-actor:' || b."id"), 1, 16))
FROM "business_units" b
WHERE NOT EXISTS (SELECT 1 FROM "system_actors" sa WHERE sa."business_unit_id" = b."id")
    AND EXISTS (
        SELECT 1 FROM "users" u
        WHERE u."id" = 'usr_' || substr(substr(b."id", 4), 1, 10) || upper(substr(md5('system-actor:' || b."id"), 1, 16))
    );

UPDATE "users" u
SET "username" = 'system-' || lower(right(u."business_unit_id", 13)),
    "email_address" = 'system-' || lower(right(u."business_unit_id", 13)) || '@system.trenova.invalid',
    "password" = '!',
    "name" = 'System Account'
FROM "system_actors" sa
WHERE sa."user_id" = u."id";
