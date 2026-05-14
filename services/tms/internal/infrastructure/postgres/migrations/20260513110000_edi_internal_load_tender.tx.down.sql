DROP INDEX IF EXISTS "idx_edi_load_tender_transfers_outbound";
DROP INDEX IF EXISTS "idx_edi_load_tender_transfers_inbound";
DROP INDEX IF EXISTS "idx_edi_load_tender_transfers_open_unique";
DROP TABLE IF EXISTS "edi_load_tender_transfers";

DROP INDEX IF EXISTS "idx_edi_mapping_profile_items_target";
DROP INDEX IF EXISTS "idx_edi_mapping_profile_items_unique";
DROP TABLE IF EXISTS "edi_mapping_profile_items";

DROP INDEX IF EXISTS "idx_edi_mapping_profiles_partner";
DROP TABLE IF EXISTS "edi_mapping_profiles";

DROP TYPE IF EXISTS edi_load_tender_transfer_status_enum;
DROP TYPE IF EXISTS edi_mapping_entity_type_enum;
