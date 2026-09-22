-- Enum values are added outside a transaction because PostgreSQL refuses to
-- use a value added in the same transaction that added it.
ALTER TYPE "agent_subject_type_enum" ADD VALUE IF NOT EXISTS 'DetentionOccurrence';

--bun:split

ALTER TYPE "agent_subject_type_enum" ADD VALUE IF NOT EXISTS 'Worker';

--bun:split

ALTER TYPE "agent_subject_type_enum" ADD VALUE IF NOT EXISTS 'CarrierIntelEvent';

--bun:split

ALTER TYPE "agent_subject_type_enum" ADD VALUE IF NOT EXISTS 'EDIInboundFile';
