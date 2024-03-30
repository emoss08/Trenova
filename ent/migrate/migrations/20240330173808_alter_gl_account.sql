-- Modify "tags" table
ALTER TABLE "tags" DROP COLUMN "general_ledger_account_tags";
-- Create "general_ledger_account_tags" table
CREATE TABLE "general_ledger_account_tags" ("general_ledger_account_id" uuid NOT NULL, "tag_id" uuid NOT NULL, PRIMARY KEY ("general_ledger_account_id", "tag_id"), CONSTRAINT "general_ledger_account_tags_general_ledger_account_id" FOREIGN KEY ("general_ledger_account_id") REFERENCES "general_ledger_accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "general_ledger_account_tags_tag_id" FOREIGN KEY ("tag_id") REFERENCES "tags" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
