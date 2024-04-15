-- Modify "tags" table
ALTER TABLE "tags" ADD COLUMN "general_ledger_account_tags" uuid NULL, ADD CONSTRAINT "tags_general_ledger_accounts_tags" FOREIGN KEY ("general_ledger_account_tags") REFERENCES "general_ledger_accounts" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
