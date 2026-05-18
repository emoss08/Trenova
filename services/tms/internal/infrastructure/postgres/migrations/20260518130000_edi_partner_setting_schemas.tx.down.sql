ALTER TABLE "edi_partner_document_profiles"
    DROP CONSTRAINT IF EXISTS "fk_edi_partner_document_profiles_partner_settings_schema";

DROP INDEX IF EXISTS "idx_edi_partner_document_profiles_partner_settings_schema";

ALTER TABLE "edi_partner_document_profiles"
    DROP COLUMN IF EXISTS "partner_settings_schema_version",
    DROP COLUMN IF EXISTS "partner_settings_schema_id";

DROP TABLE IF EXISTS "edi_partner_setting_fields";
DROP TABLE IF EXISTS "edi_partner_setting_schemas";
DROP TYPE IF EXISTS edi_partner_setting_status_enum;
DROP TYPE IF EXISTS edi_partner_setting_data_type_enum;
