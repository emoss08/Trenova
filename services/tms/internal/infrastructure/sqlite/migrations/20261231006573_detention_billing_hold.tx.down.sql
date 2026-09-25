-- Hand-completed mirror of 20261231006573_detention_billing_hold.tx.down.sql.

DROP INDEX IF EXISTS "idx_detention_occurrences_billing_hold";

--bun:split

DROP INDEX IF EXISTS "idx_invoice_lines_additional_charge";

--bun:split

ALTER TABLE "invoice_lines" DROP COLUMN "additional_charge_id";
