-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231003200_agent_core_tools.tx.up.sql
--
-- Hand-completed: SQLite keeps tool_names as the array's text literal, with no
-- array_remove to take names out of it. The cleanup is cosmetic — a core tool
-- still listed is held once either way, and the next save of the agent drops
-- it — so the SQLite side leaves the rows as they are.

SELECT 1;
