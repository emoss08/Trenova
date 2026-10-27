--keel:notx
-- Enum values are added outside a transaction because PostgreSQL refuses to
-- use a value added in the same transaction that added it. A notice of
-- assignment is the record an expiring-notice conversation is about.
ALTER TYPE "agent_subject_type_enum" ADD VALUE IF NOT EXISTS 'FactoringNoa';
