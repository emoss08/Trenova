-- Drop EDI profiles table and related indexes

DROP INDEX IF EXISTS edi_profiles_bu_org_id_idx;
DROP INDEX IF EXISTS edi_profiles_bu_org_name_id_idx;
DROP INDEX IF EXISTS edi_profiles_bu_org_name_uidx;

DROP TABLE IF EXISTS edi_profiles;

