-- Enum values are added outside a transaction because PostgreSQL refuses to
-- use a value added in the same transaction that added it. An invoice and a
-- dispute are records a Desk conversation can be a case about.
ALTER TYPE "agent_subject_type_enum" ADD VALUE IF NOT EXISTS 'Invoice';

--bun:split
ALTER TYPE "agent_subject_type_enum" ADD VALUE IF NOT EXISTS 'InvoiceDispute';
