-- Modify "general_ledger_accounts" table
ALTER TABLE "general_ledger_accounts" ALTER COLUMN "status" TYPE character varying(1), ALTER COLUMN "account_number" TYPE character varying(7), ALTER COLUMN "account_type" TYPE character varying(9), ALTER COLUMN "date_opened" TYPE date, ALTER COLUMN "date_closed" TYPE date;
