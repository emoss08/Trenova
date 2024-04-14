-- Create "us_states" table
CREATE TABLE "us_states" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "name" character varying NOT NULL, "abbreviation" character varying NOT NULL, "country_name" character varying NOT NULL DEFAULT 'United States', "country_iso3" character varying NOT NULL DEFAULT 'USA', PRIMARY KEY ("id"));
