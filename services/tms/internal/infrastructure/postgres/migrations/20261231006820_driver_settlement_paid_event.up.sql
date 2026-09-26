-- Enum values are added outside a transaction because PostgreSQL refuses to
-- use a value added in the same transaction that added it.
ALTER TYPE journal_source_event_enum ADD VALUE IF NOT EXISTS 'DriverSettlementPaid';
