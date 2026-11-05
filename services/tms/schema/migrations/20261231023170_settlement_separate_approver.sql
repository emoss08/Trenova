-- Whoever submits a driver settlement cannot also approve it. On by default:
-- pay is the money an organization most needs a second person on, and a
-- one-person payroll office turns this off in Settlement Control.
ALTER TABLE "settlement_controls"
    ADD COLUMN IF NOT EXISTS "require_separate_approver" BOOLEAN NOT NULL DEFAULT TRUE;
