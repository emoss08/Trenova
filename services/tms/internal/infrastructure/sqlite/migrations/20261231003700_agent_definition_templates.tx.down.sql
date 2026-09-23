-- Hand-written: the narrower template check is not restored. Rows written
-- under the wider one would fail it, and a rollback that deletes agents is
-- worse than a check that allows templates the code no longer offers.
-- Source: 20261231003700_agent_definition_templates.tx.down.sql

SELECT 1;
