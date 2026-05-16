DROP INDEX IF EXISTS "idx_edi_transfer_changes_search";
DROP TABLE IF EXISTS "edi_transfer_changes";

DROP INDEX IF EXISTS "idx_edi_shipment_links_target_lookup";
DROP INDEX IF EXISTS "idx_edi_shipment_links_source_lookup";
DROP INDEX IF EXISTS "idx_edi_shipment_links_source_target";
DROP INDEX IF EXISTS "idx_edi_shipment_links_transfer";
DROP TABLE IF EXISTS "edi_shipment_links";

DROP INDEX IF EXISTS "idx_shipments_entry_method";
DROP INDEX IF EXISTS "idx_shipments_tender_status";
ALTER TABLE "shipments"
    DROP COLUMN IF EXISTS "entry_method",
    DROP COLUMN IF EXISTS "tender_status";

DROP TYPE IF EXISTS edi_transfer_change_conflict_status_enum;
DROP TYPE IF EXISTS edi_transfer_change_status_enum;
DROP TYPE IF EXISTS edi_transfer_change_direction_enum;
DROP TYPE IF EXISTS edi_shipment_link_status_enum;
DROP TYPE IF EXISTS edi_shipment_sync_policy_enum;
DROP TYPE IF EXISTS shipment_entry_method_enum;
DROP TYPE IF EXISTS shipment_tender_status_enum;

ALTER TABLE "edi_partners"
    DROP CONSTRAINT IF EXISTS "fk_edi_partners_default_transport";

ALTER TABLE "edi_partners"
    DROP CONSTRAINT IF EXISTS "fk_edi_partners_connection";

ALTER TABLE "edi_partners"
    DROP COLUMN IF EXISTS "edi_connection_id";

DROP INDEX IF EXISTS "idx_edi_communication_profiles_partner";
DROP INDEX IF EXISTS "idx_edi_communication_profiles_search";
DROP INDEX IF EXISTS "idx_edi_communication_profiles_name_org";
DROP TABLE IF EXISTS "edi_communication_profiles";

DROP INDEX IF EXISTS "idx_edi_connections_target";
DROP INDEX IF EXISTS "idx_edi_connections_source";
DROP INDEX IF EXISTS "idx_edi_connections_internal_open";
DROP TABLE IF EXISTS "edi_connections";

DROP TYPE IF EXISTS edi_connection_status_enum;
DROP TYPE IF EXISTS edi_connection_method_enum;
