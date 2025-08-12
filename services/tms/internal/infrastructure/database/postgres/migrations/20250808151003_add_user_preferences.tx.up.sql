CREATE TABLE IF NOT EXISTS "user_preferences"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "auto_shipment_ownership" boolean NOT NULL DEFAULT TRUE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_user_preferences" PRIMARY KEY ("id", "business_unit_id", "organization_id", "user_id"),
    CONSTRAINT "fk_user_preferences_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_user_preferences_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_preferences_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uk_user_preferences_user" UNIQUE ("user_id")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_user_preferences_org_bu_user" ON "user_preferences"("organization_id", "business_unit_id", "user_id");

CREATE INDEX IF NOT EXISTS "idx_user_preferences_created_updated" ON "user_preferences"("created_at", "updated_at");

COMMENT ON TABLE "user_preferences" IS 'Stores user preferences';

