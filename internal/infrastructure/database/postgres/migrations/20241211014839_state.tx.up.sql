-- Create the table for US states with comments for each column
CREATE TABLE IF NOT EXISTS "us_states"(
    "id" varchar(100), -- Unique identifier for the us state
    "name" varchar(100) NOT NULL, -- Name of the us state
    "abbreviation" varchar(7) NOT NULL, -- Abbreviation of the us state
    "country_name" varchar(100) NOT NULL DEFAULT 'United States', -- Name of the country where the us state is located
    "country_iso3" varchar(3) NOT NULL DEFAULT 'USA', -- ISO3 code of the country where the us state is located
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    PRIMARY KEY ("id") -- Primary key constraint
);

--bun:split
-- Create an index on the name column
CREATE UNIQUE INDEX "idx_us_states_name" ON "us_states"("name");

CREATE UNIQUE INDEX "idx_us_states_abbreviation" ON "us_states"("abbreviation");

--bun:split
-- Add comments for each column
COMMENT ON COLUMN "us_states"."name" IS 'Name of the us state';

COMMENT ON COLUMN "us_states"."abbreviation" IS 'Abbreviation of the us state, must be 4 characters';

COMMENT ON COLUMN "us_states"."country_name" IS 'Name of the country where the us state is located, defaults to United States';

COMMENT ON COLUMN "us_states"."country_iso3" IS 'ISO3 code of the country where the us state is located, defaults to USA';

COMMENT ON COLUMN "us_states"."created_at" IS 'Timestamp when the us state was created, defaults to current timestamp';

COMMENT ON COLUMN "us_states"."updated_at" IS 'Timestamp when the us state was last updated, defaults to current timestamp';

