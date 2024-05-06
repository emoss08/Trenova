-- Modify "users" table
ALTER TABLE "users" ALTER COLUMN "timezone" TYPE character varying(20), ALTER COLUMN "timezone" DROP DEFAULT;
