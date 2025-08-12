CREATE TABLE IF NOT EXISTS "edi_profiles"(
    "id" varchar(100) PRIMARY KEY,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(255) NOT NULL,
    "schema_path" varchar(255) NOT NULL,
    "delims" jsonb NOT NULL DEFAULT '{}' ::jsonb,
    "validation" jsonb NOT NULL DEFAULT '{}' ::jsonb,
    "references" jsonb NOT NULL DEFAULT '{}' ::jsonb,
    "party_roles" jsonb NOT NULL DEFAULT '{}' ::jsonb,
    "stop_type_map" jsonb NOT NULL DEFAULT '{}' ::jsonb,
    "shipment_id_quals" jsonb NOT NULL DEFAULT '[]' ::jsonb,
    "shipment_id_mode" varchar(255) NOT NULL DEFAULT 'ref_first',
    "carrier_scac_fallback" varchar(255) NOT NULL DEFAULT '',
    "include_raw_l11" boolean NOT NULL DEFAULT FALSE,
    "raw_l11_filter" jsonb NOT NULL DEFAULT '[]' ::jsonb,
    "equipment_type_map" jsonb NOT NULL DEFAULT '{}' ::jsonb,
    "include_segments" boolean NOT NULL DEFAULT FALSE,
    "emit_iso_datetime" boolean NOT NULL DEFAULT FALSE,
    "timezone" varchar(255) NOT NULL DEFAULT 'UTC',
    "service_level_quals" jsonb NOT NULL DEFAULT '[]' ::jsonb,
    "service_level_map" jsonb NOT NULL DEFAULT '{}' ::jsonb,
    "accessorial_quals" jsonb NOT NULL DEFAULT '[]' ::jsonb,
    "accessorial_map" jsonb NOT NULL DEFAULT '{}' ::jsonb,
    "version" bigint,
    "created_at" bigint NOT NULL DEFAULT (extract(epoch FROM current_timestamp)) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT (extract(epoch FROM current_timestamp)) ::bigint,
    CONSTRAINT "fk_edi_profiles_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_edi_profiles_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "edi_profiles_bu_org_name_uidx" ON "edi_profiles"("business_unit_id", "organization_id", "name");

CREATE INDEX IF NOT EXISTS "edi_profiles_bu_org_name_id_idx" ON "edi_profiles"("business_unit_id", "organization_id", "name", "id");

CREATE INDEX IF NOT EXISTS "edi_profiles_bu_org_id_idx" ON "edi_profiles"("business_unit_id", "organization_id", "id");

COMMENT ON TABLE "edi_profiles" IS 'Partner-specific EDI configuration profiles (delimiters, mapping, validation)';

COMMENT ON COLUMN "edi_profiles"."delims" IS 'Delimiters: element, component, segment, repetition';

COMMENT ON COLUMN "edi_profiles"."validation" IS 'Validation profile (strictness, se-count enforcement, required segments)';

COMMENT ON COLUMN "edi_profiles"."references" IS 'DTO reference mapping from L11 qualifiers';

COMMENT ON COLUMN "edi_profiles"."party_roles" IS 'Mapping from DTO roles to N1 codes';

COMMENT ON COLUMN "edi_profiles"."stop_type_map" IS 'S5 type normalization map';

COMMENT ON COLUMN "edi_profiles"."service_level_map" IS 'Normalize raw service level values to canonical names';

COMMENT ON COLUMN edi_profiles.accessorial_map IS 'Normalize accessorial codes to canonical names';

