DROP STATISTICS IF EXISTS "edi_partners_kind_status_bu_stats";
DROP STATISTICS IF EXISTS "edi_partners_kind_status_org_stats";

DROP INDEX IF EXISTS "idx_edi_partners_search";
DROP INDEX IF EXISTS "idx_edi_partners_created_updated";
DROP INDEX IF EXISTS "idx_edi_partners_internal_relationship_org_bu";
DROP INDEX IF EXISTS "idx_edi_partners_name_org";
DROP INDEX IF EXISTS "idx_edi_partners_code_org";

DROP TABLE IF EXISTS "edi_partners";

DROP TYPE IF EXISTS edi_partner_role_enum;
DROP TYPE IF EXISTS edi_partner_kind_enum;
