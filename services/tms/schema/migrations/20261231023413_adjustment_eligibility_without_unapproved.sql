-- An adjustment to a paid, partly paid or disputed invoice is either refused or
-- approved: AllowWithoutApproval leaves the type. The validator never let it
-- be saved, so a stored one is moved to the stricter AllowWithApproval.
ALTER TABLE invoice_adjustment_controls
    ALTER COLUMN partially_paid_invoice_adjustment_policy DROP DEFAULT,
    ALTER COLUMN paid_invoice_adjustment_policy DROP DEFAULT,
    ALTER COLUMN disputed_invoice_adjustment_policy DROP DEFAULT;

ALTER TYPE adjustment_eligibility_policy_enum RENAME TO adjustment_eligibility_policy_enum_old;

CREATE TYPE adjustment_eligibility_policy_enum AS ENUM ('Disallow', 'AllowWithApproval');

ALTER TABLE invoice_adjustment_controls
    ALTER COLUMN partially_paid_invoice_adjustment_policy TYPE adjustment_eligibility_policy_enum
        USING (CASE partially_paid_invoice_adjustment_policy::text
            WHEN 'AllowWithoutApproval' THEN 'AllowWithApproval'
            ELSE partially_paid_invoice_adjustment_policy::text
        END)::adjustment_eligibility_policy_enum,
    ALTER COLUMN paid_invoice_adjustment_policy TYPE adjustment_eligibility_policy_enum
        USING (CASE paid_invoice_adjustment_policy::text
            WHEN 'AllowWithoutApproval' THEN 'AllowWithApproval'
            ELSE paid_invoice_adjustment_policy::text
        END)::adjustment_eligibility_policy_enum,
    ALTER COLUMN disputed_invoice_adjustment_policy TYPE adjustment_eligibility_policy_enum
        USING (CASE disputed_invoice_adjustment_policy::text
            WHEN 'AllowWithoutApproval' THEN 'AllowWithApproval'
            ELSE disputed_invoice_adjustment_policy::text
        END)::adjustment_eligibility_policy_enum;

ALTER TABLE invoice_adjustment_controls
    ALTER COLUMN partially_paid_invoice_adjustment_policy SET DEFAULT 'AllowWithApproval',
    ALTER COLUMN paid_invoice_adjustment_policy SET DEFAULT 'Disallow',
    ALTER COLUMN disputed_invoice_adjustment_policy SET DEFAULT 'AllowWithApproval';

DROP TYPE adjustment_eligibility_policy_enum_old;
