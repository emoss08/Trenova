-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261229000500_agent_controls_shrink.tx.up.sql

ALTER TABLE "agent_controls" DROP COLUMN "billing_agent_enabled";

--bun:split

ALTER TABLE "agent_controls" DROP COLUMN "dispatch_agent_enabled";

--bun:split

ALTER TABLE "agent_controls" DROP COLUMN "dispatch_autonomy_tier";

--bun:split

ALTER TABLE "agent_controls" DROP COLUMN "decision_timeout_seconds";
