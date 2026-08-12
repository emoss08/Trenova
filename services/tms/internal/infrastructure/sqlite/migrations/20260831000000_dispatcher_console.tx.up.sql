-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260831000000_dispatcher_console.tx.up.sql

ALTER TABLE "dispatch_controls" ADD COLUMN "scoring_weights" TEXT NOT NULL DEFAULT '{}';

--bun:split

ALTER TABLE "dispatch_controls" ADD COLUMN "auto_assign_confidence_threshold" NUMERIC NOT NULL DEFAULT 0.8500;

--bun:split

ALTER TABLE "dispatch_controls" ADD COLUMN "auto_assign_max_deadhead_miles" INTEGER;

--bun:split

ALTER TABLE "dispatch_controls" ADD COLUMN "auto_assign_planning_horizon_hours" INTEGER NOT NULL DEFAULT 48;

--bun:split

ALTER TABLE "agent_controls" ADD COLUMN "dispatch_agent_enabled" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "agent_controls" ADD COLUMN "dispatch_autonomy_tier" TEXT NOT NULL DEFAULT 'Propose';
