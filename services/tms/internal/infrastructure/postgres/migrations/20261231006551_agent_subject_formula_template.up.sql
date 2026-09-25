-- Enum values are added outside a transaction because PostgreSQL refuses to
-- use a value added in the same transaction that added it.
ALTER TYPE "agent_subject_type_enum" ADD VALUE IF NOT EXISTS 'FormulaTemplate';
