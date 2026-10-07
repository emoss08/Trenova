-- Hand-written: the converter does not carry this DROP TABLE across, so the
-- reverse of the up migration is written out here.
-- Source: 20261231008450_ai_setting_versions.tx.down.sql

DROP TABLE IF EXISTS "ai_setting_versions";
