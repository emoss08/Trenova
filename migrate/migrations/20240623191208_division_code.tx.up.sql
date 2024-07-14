CREATE TABLE IF NOT EXISTS "division_codes"
(
    "id"                 uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id"   uuid        NOT NULL,
    "organization_id"    uuid        NOT NULL,
    "status"             status_enum NOT NULL DEFAULT 'Active',
    "code"               VARCHAR(4)  NOT NULL,
    "description"        TEXT,
    "color"              VARCHAR(10),
    "cash_account_id"    uuid,
    "ap_account_id"      uuid,
    "expense_account_id" uuid,
    "version"            BIGINT      NOT NULL,
    "created_at"         TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"         TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("cash_account_id") REFERENCES general_ledger_accounts ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    FOREIGN KEY ("ap_account_id") REFERENCES general_ledger_accounts ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    FOREIGN KEY ("expense_account_id") REFERENCES general_ledger_accounts ("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "division_codes_code_organization_id_unq" ON "division_codes" (LOWER("code"), organization_id);
CREATE INDEX idx_division_codes_code ON division_codes (code);
CREATE INDEX idx_division_codes_org_bu ON division_codes (organization_id, business_unit_id);
CREATE INDEX idx_division_codes_description ON division_codes USING GIN (description gin_trgm_ops);
CREATE INDEX idx_division_codes_created_at ON division_codes (created_at);

--bun:split
COMMENT ON COLUMN division_codes.id IS 'Unique identifier for the division code, generated as a UUID';
COMMENT ON COLUMN division_codes.business_unit_id IS 'Foreign key referencing the business unit that this division code belongs to';
COMMENT ON COLUMN division_codes.organization_id IS 'Foreign key referencing the organization that this division code belongs to';
COMMENT ON COLUMN division_codes.status IS 'The current status of the division code, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN division_codes.code IS 'A short, unique code for identifying the division code, limited to 4 characters';
COMMENT ON COLUMN division_codes.description IS 'A detailed description of the division code';
COMMENT ON COLUMN division_codes.color IS 'The color associated with the division code';
COMMENT ON COLUMN division_codes.cash_account_id IS 'Foreign key referencing the cash account associated with the division code';
COMMENT ON COLUMN division_codes.ap_account_id IS 'Foreign key referencing the accounts payable account associated with the division code';
COMMENT ON COLUMN division_codes.expense_account_id IS 'Foreign key referencing the expense account associated with the division code';
COMMENT ON COLUMN division_codes.created_at IS 'Timestamp of when the division code was created, defaults to the current timestamp';
COMMENT ON COLUMN division_codes.updated_at IS 'Timestamp of the last update to the division code, defaults to the current timestamp';
