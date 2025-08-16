CREATE TABLE IF NOT EXISTS "hold_reasons"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "type" hold_type_enum NOT NULL,
    "code" varchar(64) NOT NULL,
    "label" varchar(100) NOT NULL,
    "description" text,
    "default_severity" hold_severity_enum NOT NULL DEFAULT 'Advisory',
    "default_blocks_dispatch" boolean NOT NULL DEFAULT FALSE,
    "default_blocks_delivery" boolean NOT NULL DEFAULT FALSE,
    "default_blocks_billing" boolean NOT NULL DEFAULT FALSE,
    "default_visible_to_customer" boolean NOT NULL DEFAULT FALSE,
    "active" boolean NOT NULL DEFAULT TRUE,
    "sort_order" integer NOT NULL DEFAULT 100,
    "external_map" jsonb,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_hold_reasons" PRIMARY KEY ("id", "organization_id"),
    CONSTRAINT "fk_hr_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_hr_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ux_hr_org_bu_type_code" UNIQUE ("organization_id", "business_unit_id", "type", "code")
);

CREATE INDEX IF NOT EXISTS "idx_hr_org_bu_type_active" ON "hold_reasons"("organization_id", "business_unit_id", "type")
WHERE
    active = TRUE;

COMMENT ON TABLE hold_reasons IS 'Reasons for holds. Each reason can be associated with a specific hold type and severity, with default settings for blocking and visibility.';

