CREATE TABLE
    IF NOT EXISTS "pro_number_counters"
(
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "organization_id"  uuid        NOT NULL,
    "last_used_number" integer     NOT NULL DEFAULT 0,
    "updated_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    UNIQUE ("organization_id")
);
