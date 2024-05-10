-- Modify "permissions" table
ALTER TABLE "permissions" DROP COLUMN "name", DROP COLUMN "name_humanized", ADD COLUMN "codename" character varying NOT NULL, ADD COLUMN "label" character varying NULL, ADD COLUMN "read_description" character varying NULL, ADD COLUMN "write_description" character varying NULL;
