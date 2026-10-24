--keel:notx
-- Enum values are added outside a transaction because PostgreSQL refuses to
-- use a value added in the same transaction that added it. A factoring
-- schedule is the record a sent or failed-schedule conversation is about.
ALTER TYPE "agent_subject_type_enum" ADD VALUE IF NOT EXISTS 'FactoringSubmission';
