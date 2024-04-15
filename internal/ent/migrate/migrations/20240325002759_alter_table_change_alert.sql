-- Modify "table_change_alerts" table
ALTER TABLE "table_change_alerts" DROP COLUMN "topic", ADD COLUMN "topic_name" character varying NULL;
