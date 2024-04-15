-- Modify "accounting_controls" table
ALTER TABLE "accounting_controls" ADD COLUMN "journal_entry_criteria" character varying NOT NULL DEFAULT 'OnShipmentBill';
