-- Enum values are added outside a transaction because PostgreSQL refuses to
-- use a value added in the same transaction that added it. A 997 or 999 that
-- accepts a transaction set while noting errors in it (AK5 or IK5 "E") is
-- AcceptedWithErrors, so the errors the partner reported are not lost.
ALTER TYPE "edi_message_ack_status_enum" ADD VALUE IF NOT EXISTS 'AcceptedWithErrors' AFTER 'Accepted';
