DROP TABLE IF EXISTS "tags" CASCADE;

-- bun:split

DROP INDEX IF EXISTS "tags_name_organization_id_unq" CASCADE;

-- bun:split

DROP TABLE IF EXISTS "tag_general_ledger_accounts" CASCADE;
