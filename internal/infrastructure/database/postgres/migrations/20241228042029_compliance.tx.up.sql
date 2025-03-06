CREATE TABLE IF NOT EXISTS "hazmat_expirations" (
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "state_id" varchar(100) NOT NULL,
    -- Core Fields
    "years" smallint NOT NULL CHECK (years > 0),
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_hazmat_expirations" PRIMARY KEY ("id", "state_id"),
    CONSTRAINT "fk_hazmat_expirations_state" FOREIGN KEY ("state_id") REFERENCES "us_states" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);

-- Indexes
CREATE INDEX "idx_hazmat_expirations_state" ON "hazmat_expirations" ("state_id");

CREATE INDEX "idx_hazmat_expirations_years" ON "hazmat_expirations" ("years");

COMMENT ON TABLE hazmat_expirations IS 'Stores information about hazmat expirations';

