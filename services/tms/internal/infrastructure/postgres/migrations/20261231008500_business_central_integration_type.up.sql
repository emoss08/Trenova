-- Enum values are added outside a transaction because PostgreSQL refuses to
-- use a value added in the same transaction that added it.
ALTER TYPE "integration_type" ADD VALUE IF NOT EXISTS 'BusinessCentral';
