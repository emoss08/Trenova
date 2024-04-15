-- Modify "workers" table
ALTER TABLE "workers" ALTER COLUMN "city" TYPE character varying(150), ADD COLUMN "address_line_1" character varying(150) NULL, ADD COLUMN "address_line_2" character varying(150) NULL;
