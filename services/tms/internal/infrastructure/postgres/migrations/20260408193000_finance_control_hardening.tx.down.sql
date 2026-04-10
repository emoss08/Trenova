UPDATE invoice_adjustment_controls
SET
    standard_adjustment_approval_threshold = NULL
WHERE standard_adjustment_approval_policy = 'AmountThreshold'
  AND standard_adjustment_approval_threshold = 0.01;

--bun:split
UPDATE invoice_adjustment_controls
SET
    write_off_approval_threshold = NULL
WHERE write_off_approval_policy = 'RequireApprovalAboveThreshold'
  AND write_off_approval_threshold = 0.01;
