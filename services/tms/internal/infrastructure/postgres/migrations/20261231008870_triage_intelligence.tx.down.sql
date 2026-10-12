DELETE FROM "ai_corrections" WHERE "task" = 'InboundClassification';

--bun:split
ALTER TABLE "ai_corrections"
    DROP CONSTRAINT IF EXISTS "ck_ai_corrections_task",
    DROP CONSTRAINT IF EXISTS "ck_ai_corrections_source_type",
    DROP CONSTRAINT IF EXISTS "ck_ai_corrections_subject_type";

--bun:split
ALTER TABLE "ai_corrections"
    ADD CONSTRAINT "ck_ai_corrections_task" CHECK ("task" IN ('ShipmentDraftExtraction')),
    ADD CONSTRAINT "ck_ai_corrections_source_type" CHECK ("source_type" IN ('DocumentShipmentDraft')),
    ADD CONSTRAINT "ck_ai_corrections_subject_type" CHECK ("subject_type" IN ('Shipment'));

--bun:split
DROP INDEX IF EXISTS "idx_inbound_messages_received";
