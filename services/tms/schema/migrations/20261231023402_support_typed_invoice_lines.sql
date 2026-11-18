-- An invoice line whose description a person typed (a standalone credit or debit memo's lines)
-- is told apart from one derived by rating, posting or an adjustment, so Trenova support can
-- still re-rate and post while it cannot put words of its own on an invoice. The memo service
-- sets the mark; every derived line leaves it false. Memo lines already stored were typed.
ALTER TABLE "invoice_lines"
    ADD COLUMN IF NOT EXISTS "typed_description" boolean NOT NULL DEFAULT false;

UPDATE "invoice_lines"
SET "typed_description" = true
WHERE "type" = 'Memo'
    AND "typed_description" = false;

COMMENT ON COLUMN "invoice_lines"."typed_description" IS 'Whether a person typed the description, rather than rating, posting or an adjustment deriving it';

DROP TRIGGER IF EXISTS "guard_support_outbound_typed" ON "invoice_lines";

CREATE TRIGGER "guard_support_outbound_typed"
    BEFORE INSERT ON "invoice_lines"
    FOR EACH ROW
    WHEN (NEW."typed_description")
    EXECUTE FUNCTION "support_outbound_guard"();

-- An order charge's description is typed and prints on the invoice the order is billed on.
DROP TRIGGER IF EXISTS "guard_support_outbound" ON "order_charges";

CREATE TRIGGER "guard_support_outbound"
    BEFORE INSERT OR UPDATE ON "order_charges"
    FOR EACH ROW
    EXECUTE FUNCTION "support_outbound_guard"('description');
