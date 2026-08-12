-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260408193000_finance_control_hardening.tx.up.sql

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
