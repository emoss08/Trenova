DROP TABLE IF EXISTS "general_ledger_accounts" CASCADE;

-- bun:split

DROP TYPE IF EXISTS account_type_enum CASCADE;

-- bun:split

DROP TYPE IF EXISTS cash_flow_type_enum CASCADE;

-- bun:split

DROP TYPE IF EXISTS account_sub_type_enum CASCADE;

-- bun:split

DROP TYPE IF EXISTS account_classification_type_enum CASCADE;

-- bun:split

DROP INDEX IF EXISTS "gl_account_account_number_organization_id_unq" CASCADE;
