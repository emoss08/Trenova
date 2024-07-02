CREATE TABLE
    IF NOT EXISTS "revenue_codes"
(
    "id"                 uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id"   uuid        NOT NULL,
    "organization_id"    uuid        NOT NULL,
    "status"             status_enum NOT NULL DEFAULT 'Active',
    "code"               VARCHAR(10) NOT NULL,
    "description"        TEXT,
    "expense_account_id" uuid,
    "revenue_account_id" uuid,
    "color"              VARCHAR(10),
    "created_at"         TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"         TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("revenue_account_id") REFERENCES general_ledger_accounts ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    FOREIGN KEY ("expense_account_id") REFERENCES general_ledger_accounts ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "revenue_codes_code_organization_id_unq" ON "revenue_codes" (LOWER("code"), organization_id);
CREATE INDEX idx_revenue_codes_code ON revenue_codes (code);
CREATE INDEX idx_revenue_codes_org_bu ON revenue_codes (organization_id, business_unit_id);
CREATE INDEX idx_revenue_codes_description ON revenue_codes USING GIN (description gin_trgm_ops);
CREATE INDEX idx_revenue_codes_created_at ON revenue_codes(created_at);

--bun:split

COMMENT ON COLUMN revenue_codes.id IS 'Unique identifier for the revenue code, generated as a UUID';
COMMENT ON COLUMN revenue_codes.business_unit_id IS 'Foreign key referencing the business unit that this revenue code belongs to';
COMMENT ON COLUMN revenue_codes.organization_id IS 'Foreign key referencing the organization that this revenue code belongs to';
COMMENT ON COLUMN revenue_codes.status IS 'The current status of the revenue code, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN revenue_codes.code IS 'A short, unique code for identifying the revenue code, limited to 10 characters';
COMMENT ON COLUMN revenue_codes.description IS 'A detailed description of the revenue code';
COMMENT ON COLUMN revenue_codes.expense_account_id IS 'Foreign key referencing the expense account associated with this revenue code';
COMMENT ON COLUMN revenue_codes.revenue_account_id IS 'Foreign key referencing the revenue account associated with this revenue code';
COMMENT ON COLUMN revenue_codes.color IS 'The color associated with the revenue code, represented as a string limited to 10 characters';
COMMENT ON COLUMN revenue_codes.created_at IS 'Timestamp of when the revenue code was created, defaults to the current timestamp';
COMMENT ON COLUMN revenue_codes.updated_at IS 'Timestamp of the last update to the revenue code, defaults to the current timestamp';
