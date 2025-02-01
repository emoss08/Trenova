CREATE TYPE routing_provider_enum AS ENUM(
    'PCMiler'
);

CREATE TABLE IF NOT EXISTS "location_indices"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "internal_state_id" varchar(100) NOT NULL,
    -- Core fields
    "provider" routing_provider_enum NOT NULL,
    "street_address" varchar(255) NOT NULL,
    "local_area" varchar(255) NOT NULL,
    "city" varchar(255) NOT NULL,
    "state" varchar(255) NOT NULL,
    "state_name" varchar(255) NOT NULL,
    "postal_code" varchar(255) NOT NULL,
    "country" varchar(255) NOT NULL,
    "country_full_name" varchar(255) NOT NULL,
    "splc" varchar(255) NOT NULL,
    "longitude" float NOT NULL,
    "latitude" float NOT NULL,
    "short_string" varchar(255) NOT NULL,
    "time_zone" varchar(255) NOT NULL,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_location_indices" PRIMARY KEY ("id", "internal_state_id"),
    CONSTRAINT "fk_location_indices_state" FOREIGN KEY ("internal_state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);
