-- Hand-written: the converter does not carry this DROP TABLE across, so the
-- reverse of the up migration is written out here.
-- Source: 20261231008440_agent_definition_versions.tx.down.sql

DROP TABLE IF EXISTS "agent_definition_versions";
