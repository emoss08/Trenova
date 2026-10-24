--keel:notx
-- Enum values are added outside a transaction because PostgreSQL refuses to
-- use a value added in the same transaction that added it.
ALTER TYPE "integration_type" ADD VALUE IF NOT EXISTS 'QuickBooksOnline';

--keel:split
ALTER TYPE "integration_category" ADD VALUE IF NOT EXISTS 'Accounting';

--keel:split
ALTER TYPE "agent_subject_type_enum" ADD VALUE IF NOT EXISTS 'AccountingConnection';
