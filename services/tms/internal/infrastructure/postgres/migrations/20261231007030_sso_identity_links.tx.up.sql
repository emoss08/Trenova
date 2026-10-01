CREATE TABLE IF NOT EXISTS "sso_identity_links"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "sso_config_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "issuer" varchar(500) NOT NULL,
    "subject" varchar(255) NOT NULL,
    "email_at_link" varchar(320) NOT NULL,
    "last_login_at" bigint NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_sso_identity_links" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_sso_identity_links_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_sso_identity_links_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_sso_identity_links_sso_config" FOREIGN KEY ("sso_config_id", "organization_id", "business_unit_id") REFERENCES "sso_configs"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_sso_identity_links_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_sso_identity_links_subject" UNIQUE ("sso_config_id", "issuer", "subject"),
    CONSTRAINT "uq_sso_identity_links_user" UNIQUE ("sso_config_id", "issuer", "user_id")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_sso_identity_links_business_unit" ON "sso_identity_links"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_sso_identity_links_user" ON "sso_identity_links"("user_id");

--bun:split
COMMENT ON TABLE sso_identity_links IS 'Binds an identity provider subject to the Trenova user it signed in as, so later sign-ins resolve by subject rather than by a mutable email claim';
