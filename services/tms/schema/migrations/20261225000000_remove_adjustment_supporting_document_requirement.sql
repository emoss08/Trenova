-- Supporting documents are no longer required for invoice adjustments, rebills,
-- reversals or voids. Documents can still be attached to an adjustment; nothing
-- refuses one for having none. Both the organization-wide requirement and the
-- per-customer override are retired.
ALTER TABLE "customer_billing_profiles"
    DROP COLUMN IF EXISTS "invoice_adjustment_supporting_document_policy";

DROP TYPE IF EXISTS "invoice_adjustment_supporting_document_policy_enum";

ALTER TABLE "invoice_adjustment_controls"
    DROP COLUMN IF EXISTS "adjustment_attachment_requirement";

DROP TYPE IF EXISTS "adjustment_attachment_policy_enum";
