-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231003000_inbound_mailbox_secret.tx.up.sql

ALTER TABLE "inbound_mailboxes" ADD COLUMN "signing_secret" TEXT;
