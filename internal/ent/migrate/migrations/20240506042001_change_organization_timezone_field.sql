-- Modify "organizations" table
ALTER TABLE "organizations" ALTER COLUMN "timezone" TYPE character varying(20), ALTER COLUMN "timezone" DROP DEFAULT;
