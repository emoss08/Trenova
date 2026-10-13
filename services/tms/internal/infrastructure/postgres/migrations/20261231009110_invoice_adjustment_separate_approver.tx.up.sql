-- Lets an organization require that whoever submits an invoice adjustment, or
-- raises a credit memo large enough to need approval, is not the person who
-- approves or posts it. Off by default: a one-person billing office has nobody
-- else to approve.
ALTER TABLE "invoice_adjustment_controls"
    ADD COLUMN IF NOT EXISTS "require_separate_approver" BOOLEAN NOT NULL DEFAULT FALSE;
