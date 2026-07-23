CREATE TABLE IF NOT EXISTS pay_codes(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    status status_enum NOT NULL DEFAULT 'Active',
    direction VARCHAR(20) NOT NULL,
    code VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    taxable BOOLEAN NOT NULL DEFAULT TRUE,
    counts_toward_guarantee BOOLEAN NOT NULL DEFAULT TRUE,
    gl_account_id VARCHAR(100),
    default_amount_minor BIGINT,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_pay_codes PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_pay_codes_direction CHECK (direction IN ('Earning', 'Deduction')),
    CONSTRAINT chk_pay_codes_default_amount CHECK (default_amount_minor IS NULL OR default_amount_minor > 0),
    FOREIGN KEY (gl_account_id, organization_id, business_unit_id) REFERENCES gl_accounts(id, organization_id, business_unit_id) ON DELETE SET NULL
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS uq_pay_codes_org_direction_code ON pay_codes(organization_id, business_unit_id, direction, code);

--bun:split
INSERT INTO pay_codes (id, business_unit_id, organization_id, direction, code, name, taxable, is_system)
SELECT
    'payc_' || substr(md5(o.id || defs.direction || defs.code), 1, 26),
    o.business_unit_id,
    o.id,
    defs.direction,
    defs.code,
    defs.name,
    defs.taxable,
    TRUE
FROM organizations o
CROSS JOIN (
    VALUES
        ('Earning', 'PERDIEM', 'Per Diem', FALSE),
        ('Earning', 'SAFETY', 'Safety Bonus', TRUE),
        ('Earning', 'PERFORM', 'Performance Bonus', TRUE),
        ('Earning', 'LONGEVITY', 'Longevity Bonus', TRUE),
        ('Earning', 'STIPEND', 'Stipend', FALSE),
        ('Earning', 'EQUIPRENT', 'Equipment Rental', TRUE),
        ('Earning', 'OTHER', 'Other Earning', TRUE),
        ('Deduction', 'INSUR', 'Insurance', TRUE),
        ('Deduction', 'TRKLEASE', 'Truck Lease', TRUE),
        ('Deduction', 'TRLLEASE', 'Trailer Lease', TRUE),
        ('Deduction', 'ELD', 'ELD Service', TRUE),
        ('Deduction', 'FUELCARD', 'Fuel Card', TRUE),
        ('Deduction', 'ESCROW', 'Escrow Contribution', TRUE),
        ('Deduction', 'LOAN', 'Loan Repayment', TRUE),
        ('Deduction', 'OTHER', 'Other Deduction', TRUE)
) AS defs(direction, code, name, taxable)
ON CONFLICT (organization_id, business_unit_id, direction, code) DO NOTHING;

--bun:split
ALTER TABLE recurring_earnings
    ADD COLUMN IF NOT EXISTS pay_code_id VARCHAR(100);

--bun:split
UPDATE recurring_earnings re
SET pay_code_id = pc.id
FROM pay_codes pc
WHERE pc.organization_id = re.organization_id
    AND pc.business_unit_id = re.business_unit_id
    AND pc.direction = 'Earning'
    AND pc.code = CASE re.type
        WHEN 'PerDiem' THEN 'PERDIEM'
        WHEN 'SafetyBonus' THEN 'SAFETY'
        WHEN 'PerformanceBonus' THEN 'PERFORM'
        WHEN 'LongevityBonus' THEN 'LONGEVITY'
        WHEN 'Stipend' THEN 'STIPEND'
        WHEN 'EquipmentRental' THEN 'EQUIPRENT'
        ELSE 'OTHER'
    END
    AND re.pay_code_id IS NULL;

--bun:split
ALTER TABLE recurring_earnings
    ALTER COLUMN pay_code_id SET NOT NULL;

--bun:split
ALTER TABLE recurring_earnings
    ADD CONSTRAINT fk_recurring_earnings_pay_code
    FOREIGN KEY (pay_code_id, organization_id, business_unit_id)
    REFERENCES pay_codes(id, organization_id, business_unit_id) ON DELETE RESTRICT;

--bun:split
ALTER TABLE recurring_earnings
    DROP COLUMN IF EXISTS type;

--bun:split
ALTER TABLE recurring_deductions
    ADD COLUMN IF NOT EXISTS pay_code_id VARCHAR(100);

--bun:split
UPDATE recurring_deductions rd
SET pay_code_id = pc.id
FROM pay_codes pc
WHERE pc.organization_id = rd.organization_id
    AND pc.business_unit_id = rd.business_unit_id
    AND pc.direction = 'Deduction'
    AND pc.code = CASE rd.type
        WHEN 'Insurance' THEN 'INSUR'
        WHEN 'TruckLease' THEN 'TRKLEASE'
        WHEN 'TrailerLease' THEN 'TRLLEASE'
        WHEN 'ELDService' THEN 'ELD'
        WHEN 'FuelCard' THEN 'FUELCARD'
        WHEN 'EscrowContribution' THEN 'ESCROW'
        WHEN 'LoanRepayment' THEN 'LOAN'
        ELSE 'OTHER'
    END
    AND rd.pay_code_id IS NULL;

--bun:split
ALTER TABLE recurring_deductions
    ALTER COLUMN pay_code_id SET NOT NULL;

--bun:split
ALTER TABLE recurring_deductions
    ADD CONSTRAINT fk_recurring_deductions_pay_code
    FOREIGN KEY (pay_code_id, organization_id, business_unit_id)
    REFERENCES pay_codes(id, organization_id, business_unit_id) ON DELETE RESTRICT;

--bun:split
ALTER TABLE recurring_deductions
    DROP COLUMN IF EXISTS type;

--bun:split
ALTER TABLE driver_settlement_lines
    ADD COLUMN IF NOT EXISTS pay_code_id VARCHAR(100);

--bun:split
UPDATE driver_settlement_lines l
SET pay_code_id = re.pay_code_id
FROM recurring_earnings re
WHERE re.id = l.recurring_earning_id
    AND re.organization_id = l.organization_id
    AND re.business_unit_id = l.business_unit_id
    AND l.pay_code_id IS NULL;

--bun:split
UPDATE driver_settlement_lines l
SET pay_code_id = rd.pay_code_id
FROM recurring_deductions rd
WHERE rd.id = l.recurring_deduction_id
    AND rd.organization_id = l.organization_id
    AND rd.business_unit_id = l.business_unit_id
    AND l.pay_code_id IS NULL;
