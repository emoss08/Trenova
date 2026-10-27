-- 20261231009520 seeded these rows with a bare ON CONFLICT DO NOTHING, which also
-- skipped a row silently when another id already held its code or its catalog
-- slot, leaving the id the application looks up missing. Naming the target makes
-- that case fail with the constraint it broke, and restores any row that is gone.
INSERT INTO "edi_document_types"("id", "code", "name", "standard", "transaction_set", "transaction_set_id", "direction", "default_version", "status")
VALUES
    ('edidt_x12_204_inbound', 'X12-204-IN', 'X12 204 Motor Carrier Load Tender', 'X12', '204', 'edits_x12_204', 'Inbound', '004010', 'Active'),
    ('edidt_x12_990_outbound', 'X12-990-OUT', 'X12 990 Response to Load Tender', 'X12', '990', 'edits_x12_990', 'Outbound', '004010', 'Active'),
    ('edidt_x12_214_outbound', 'X12-214-OUT', 'X12 214 Shipment Status', 'X12', '214', 'edits_x12_214', 'Outbound', '004010', 'Active'),
    ('edidt_x12_210_outbound', 'X12-210-OUT', 'X12 210 Freight Invoice', 'X12', '210', 'edits_x12_210', 'Outbound', '004010', 'Active'),
    ('edidt_x12_997_outbound', 'X12-997-OUT', 'X12 997 Functional Acknowledgment', 'X12', '997', 'edits_x12_997', 'Outbound', '004010', 'Active'),
    ('edidt_x12_999_outbound', 'X12-999-OUT', 'X12 999 Implementation Acknowledgment', 'X12', '999', 'edits_x12_999', 'Outbound', '005010', 'Active')
ON CONFLICT ("id") DO NOTHING;
