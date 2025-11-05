SET statement_timeout = 0;

DROP TRIGGER IF EXISTS gl_accounts_search_update ON "gl_accounts";

DROP FUNCTION IF EXISTS gl_accounts_search_trigger() CASCADE;

DROP TABLE IF EXISTS "gl_accounts" CASCADE;
