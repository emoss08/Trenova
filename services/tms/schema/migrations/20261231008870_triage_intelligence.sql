CREATE INDEX IF NOT EXISTS "idx_inbound_messages_sender_received" ON "inbound_messages"("organization_id", "business_unit_id", "sender_domain", "received_at" DESC);

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_received" ON "inbound_messages"("organization_id", "business_unit_id", "received_at" DESC);

ALTER TABLE "ai_corrections"
    DROP CONSTRAINT IF EXISTS "ck_ai_corrections_task",
    DROP CONSTRAINT IF EXISTS "ck_ai_corrections_source_type",
    DROP CONSTRAINT IF EXISTS "ck_ai_corrections_subject_type";

ALTER TABLE "ai_corrections"
    ADD CONSTRAINT "ck_ai_corrections_task" CHECK ("task" IN ('ShipmentDraftExtraction', 'InboundClassification')),
    ADD CONSTRAINT "ck_ai_corrections_source_type" CHECK ("source_type" IN ('DocumentShipmentDraft', 'InboundMessage')),
    ADD CONSTRAINT "ck_ai_corrections_subject_type" CHECK ("subject_type" IN ('Shipment', 'InboundMessage'));

COMMENT ON INDEX "idx_inbound_messages_received" IS 'What arrived in a window, for the triage briefing, the corrections figure and the rule previews';
