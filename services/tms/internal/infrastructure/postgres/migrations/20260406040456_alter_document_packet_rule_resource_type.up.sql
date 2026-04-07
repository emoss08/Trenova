-- Change the resource type to a varchar(100)
ALTER TABLE "document_packet_rules"
    ALTER COLUMN "resource_type" SET DATA TYPE varchar(100);

--bun:split
DROP TYPE IF EXISTS "document_packet_rule_resource_type" CASCADE;

