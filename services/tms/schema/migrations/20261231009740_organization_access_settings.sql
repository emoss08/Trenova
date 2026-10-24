-- How an organization's people take up their roles. Without a row an
-- organization uses the defaults: every role a person holds is in force from
-- the moment they sign in, and a role assigned later takes effect at once.
-- An organization that wants people to choose the roles they work under for a
-- session, as dynamic separation of duty needs, turns role activation on.
CREATE TABLE IF NOT EXISTS "organization_access_settings"(
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "require_role_activation" boolean NOT NULL DEFAULT FALSE,
    "updated_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_organization_access_settings" PRIMARY KEY ("organization_id", "business_unit_id"),
    CONSTRAINT "fk_organization_access_settings_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_organization_access_settings_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- Organizations that already keep dynamic separation of duty constraints need
-- activation to enforce them, so they keep it.
INSERT INTO "organization_access_settings"("organization_id", "business_unit_id", "require_role_activation")
SELECT DISTINCT rc."organization_id", rc."business_unit_id", TRUE
FROM "role_constraints" rc
WHERE rc."type" = 'dsd'
ON CONFLICT ("organization_id", "business_unit_id") DO UPDATE
SET "require_role_activation" = TRUE;

COMMENT ON TABLE organization_access_settings IS 'How an organization''s people take up their roles. No row means every held role is in force at sign-in.';

SELECT trenova_rls.reconcile();
