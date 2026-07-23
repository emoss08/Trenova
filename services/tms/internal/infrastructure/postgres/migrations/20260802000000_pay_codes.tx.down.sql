ALTER TABLE driver_settlement_lines
    DROP COLUMN IF EXISTS pay_code_id;

--bun:split
ALTER TABLE recurring_deductions
    ADD COLUMN IF NOT EXISTS type VARCHAR(50);

--bun:split
UPDATE recurring_deductions rd
SET type = CASE pc.code
    WHEN 'INSUR' THEN 'Insurance'
    WHEN 'TRKLEASE' THEN 'TruckLease'
    WHEN 'TRLLEASE' THEN 'TrailerLease'
    WHEN 'ELD' THEN 'ELDService'
    WHEN 'FUELCARD' THEN 'FuelCard'
    WHEN 'ESCROW' THEN 'EscrowContribution'
    WHEN 'LOAN' THEN 'LoanRepayment'
    ELSE 'Other'
END
FROM pay_codes pc
WHERE pc.id = rd.pay_code_id
    AND pc.organization_id = rd.organization_id
    AND pc.business_unit_id = rd.business_unit_id;

--bun:split
ALTER TABLE recurring_deductions
    DROP CONSTRAINT IF EXISTS fk_recurring_deductions_pay_code;

--bun:split
ALTER TABLE recurring_deductions
    DROP COLUMN IF EXISTS pay_code_id;

--bun:split
ALTER TABLE recurring_earnings
    ADD COLUMN IF NOT EXISTS type VARCHAR(50);

--bun:split
UPDATE recurring_earnings re
SET type = CASE pc.code
    WHEN 'PERDIEM' THEN 'PerDiem'
    WHEN 'SAFETY' THEN 'SafetyBonus'
    WHEN 'PERFORM' THEN 'PerformanceBonus'
    WHEN 'LONGEVITY' THEN 'LongevityBonus'
    WHEN 'STIPEND' THEN 'Stipend'
    WHEN 'EQUIPRENT' THEN 'EquipmentRental'
    ELSE 'Other'
END
FROM pay_codes pc
WHERE pc.id = re.pay_code_id
    AND pc.organization_id = re.organization_id
    AND pc.business_unit_id = re.business_unit_id;

--bun:split
ALTER TABLE recurring_earnings
    DROP CONSTRAINT IF EXISTS fk_recurring_earnings_pay_code;

--bun:split
ALTER TABLE recurring_earnings
    DROP COLUMN IF EXISTS pay_code_id;

--bun:split
DROP TABLE IF EXISTS pay_codes;
