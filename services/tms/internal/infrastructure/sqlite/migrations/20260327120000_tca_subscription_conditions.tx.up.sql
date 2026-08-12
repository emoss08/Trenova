-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260327120000_tca_subscription_conditions.tx.up.sql

ALTER TABLE "tca_subscriptions" ADD COLUMN "conditions" TEXT NOT NULL DEFAULT '[]';

--bun:split

ALTER TABLE "tca_subscriptions" ADD COLUMN "condition_match" TEXT NOT NULL DEFAULT 'all';

--bun:split

ALTER TABLE "tca_subscriptions" ADD COLUMN "watched_columns" TEXT NOT NULL DEFAULT '[]';

--bun:split

ALTER TABLE "tca_subscriptions" ADD COLUMN "custom_title" TEXT NOT NULL DEFAULT '';

--bun:split

ALTER TABLE "tca_subscriptions" ADD COLUMN "custom_message" TEXT NOT NULL DEFAULT '';

--bun:split

ALTER TABLE "tca_subscriptions" ADD COLUMN "topic" TEXT NOT NULL DEFAULT '';

--bun:split

ALTER TABLE "tca_subscriptions" ADD COLUMN "priority" TEXT NOT NULL DEFAULT 'medium';
