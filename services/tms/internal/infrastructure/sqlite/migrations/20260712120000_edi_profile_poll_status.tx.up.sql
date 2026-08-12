-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260712120000_edi_profile_poll_status.tx.up.sql

ALTER TABLE "edi_communication_profiles" ADD COLUMN "last_poll_attempt_at" INTEGER;

--bun:split

ALTER TABLE "edi_communication_profiles" ADD COLUMN "last_poll_success_at" INTEGER;

--bun:split

ALTER TABLE "edi_communication_profiles" ADD COLUMN "last_poll_error" TEXT;
