-- Hand-completed mirror of 20261220000000_invoice_line_charge_detail.tx.down.sql.

DROP INDEX IF EXISTS "idx_invoice_lines_accessorial_charge";

--bun:split

ALTER TABLE "invoice_lines" DROP COLUMN "formula_template_name";

--bun:split

ALTER TABLE "invoice_lines" DROP COLUMN "rate_basis_amount";

--bun:split

ALTER TABLE "invoice_lines" DROP COLUMN "rate";

--bun:split

ALTER TABLE "invoice_lines" DROP COLUMN "rate_unit";

--bun:split

ALTER TABLE "invoice_lines" DROP COLUMN "charge_method";

--bun:split

ALTER TABLE "invoice_lines" DROP COLUMN "charge_code";

--bun:split

ALTER TABLE "invoice_lines" DROP COLUMN "accessorial_charge_id";
