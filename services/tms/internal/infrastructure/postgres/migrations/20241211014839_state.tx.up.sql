CREATE TABLE IF NOT EXISTS "us_states"
(
    "id"           varchar(100),
    "name"         varchar(100) NOT NULL,
    "abbreviation" varchar(7)   NOT NULL,
    "country_name" varchar(100) NOT NULL DEFAULT 'United States',
    "country_iso3" varchar(3)   NOT NULL DEFAULT 'USA',
    "created_at"   bigint       NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at"   bigint       NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    PRIMARY KEY ("id")
);

--bun:split
CREATE UNIQUE INDEX "idx_us_states_name" ON "us_states" ("name");

CREATE UNIQUE INDEX "idx_us_states_abbreviation" ON "us_states" ("abbreviation");

--bun:split
COMMENT ON COLUMN "us_states"."name" IS 'Name of the us state';

COMMENT ON COLUMN "us_states"."abbreviation" IS 'Abbreviation of the us state, must be 4 characters';

COMMENT ON COLUMN "us_states"."country_name" IS 'Name of the country where the us state is located, defaults to United States';

COMMENT ON COLUMN "us_states"."country_iso3" IS 'ISO3 code of the country where the us state is located, defaults to USA';

COMMENT ON COLUMN "us_states"."created_at" IS 'Timestamp when the us state was created, defaults to current timestamp';

COMMENT ON COLUMN "us_states"."updated_at" IS 'Timestamp when the us state was last updated, defaults to current timestamp';

