-- The catalog held 204 only as outbound and every other set only as inbound, so
-- an outbound 214, 210, 990, 997 or 999 template could not be made (its document
-- type was "not seeded") and an inbound 204 profile had no type to point at.
-- Every set now exists in both directions.
INSERT INTO "edi_document_types"("id", "code", "name", "standard", "transaction_set", "transaction_set_id", "direction", "default_version", "status")
VALUES
    ('edidt_x12_204_inbound', 'X12-204-IN', 'X12 204 Motor Carrier Load Tender', 'X12', '204', 'edits_x12_204', 'Inbound', '004010', 'Active'),
    ('edidt_x12_990_outbound', 'X12-990-OUT', 'X12 990 Response to Load Tender', 'X12', '990', 'edits_x12_990', 'Outbound', '004010', 'Active'),
    ('edidt_x12_214_outbound', 'X12-214-OUT', 'X12 214 Shipment Status', 'X12', '214', 'edits_x12_214', 'Outbound', '004010', 'Active'),
    ('edidt_x12_210_outbound', 'X12-210-OUT', 'X12 210 Freight Invoice', 'X12', '210', 'edits_x12_210', 'Outbound', '004010', 'Active'),
    ('edidt_x12_997_outbound', 'X12-997-OUT', 'X12 997 Functional Acknowledgment', 'X12', '997', 'edits_x12_997', 'Outbound', '004010', 'Active'),
    ('edidt_x12_999_outbound', 'X12-999-OUT', 'X12 999 Implementation Acknowledgment', 'X12', '999', 'edits_x12_999', 'Outbound', '005010', 'Active')
ON CONFLICT DO NOTHING;
