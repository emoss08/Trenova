DROP INDEX IF EXISTS idx_fiscal_years_unique_current;

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS idx_fiscal_years_unique_current
    ON fiscal_years(organization_id, business_unit_id)
    WHERE is_current = TRUE;

--bun:split
CREATE OR REPLACE FUNCTION enforce_single_current_fiscal_year()
    RETURNS TRIGGER
    AS $$
BEGIN
    IF NEW.is_current = TRUE THEN
        UPDATE fiscal_years
        SET is_current = FALSE
        WHERE organization_id = NEW.organization_id
            AND business_unit_id = NEW.business_unit_id
            AND id != NEW.id
            AND is_current = TRUE;
    END IF;

    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_org_memberships_one_default_per_bu
    ON user_organization_memberships(user_id, business_unit_id)
    WHERE is_default = TRUE;

--bun:split
ALTER TABLE fiscal_years
    DROP CONSTRAINT IF EXISTS fiscal_years_no_overlap_per_bu;

--bun:split
ALTER TABLE fiscal_years
    ADD CONSTRAINT fiscal_years_no_overlap_per_bu
    EXCLUDE USING gist(
        organization_id WITH =,
        business_unit_id WITH =,
        int8range(start_date, end_date, '[]') WITH &&
    );
