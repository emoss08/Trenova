UPDATE invoice_adjustment_controls
SET
    standard_adjustment_approval_threshold = CASE
        WHEN standard_adjustment_approval_policy = 'AmountThreshold'
            AND (
                standard_adjustment_approval_threshold IS NULL
                OR standard_adjustment_approval_threshold <= 0
            )
            THEN 0.01
        ELSE standard_adjustment_approval_threshold
    END,
    write_off_approval_threshold = CASE
        WHEN write_off_approval_policy = 'RequireApprovalAboveThreshold'
            AND (
                write_off_approval_threshold IS NULL
                OR write_off_approval_threshold <= 0
            )
            THEN 0.01
        ELSE write_off_approval_threshold
    END;

--bun:split
INSERT INTO invoice_adjustment_controls (
    id,
    business_unit_id,
    organization_id,
    standard_adjustment_approval_threshold,
    write_off_approval_threshold
)
SELECT
    CONCAT('iac_', replace(gen_random_uuid()::text, '-', '')) AS id,
    seed.business_unit_id,
    seed.organization_id,
    0.01,
    0.01
FROM (
    SELECT DISTINCT ON (organization_id)
        organization_id,
        business_unit_id
    FROM (
        SELECT organization_id, business_unit_id FROM billing_controls
        UNION ALL
        SELECT organization_id, business_unit_id FROM accounting_controls
    ) sources
    ORDER BY organization_id, business_unit_id
) seed
WHERE NOT EXISTS (
    SELECT 1
    FROM invoice_adjustment_controls iac
    WHERE iac.organization_id = seed.organization_id
);
