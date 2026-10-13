ALTER TABLE "factoring_companies"
    ADD COLUMN IF NOT EXISTS "non_recourse" BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN "factoring_companies"."non_recourse" IS 'True when the factor buys invoices without recourse and carries the credit risk: its invoices leave collections and credit exposure.';
