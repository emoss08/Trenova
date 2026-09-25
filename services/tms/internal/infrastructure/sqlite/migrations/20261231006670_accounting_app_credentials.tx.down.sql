-- Hand-written: SQLite has no ALTER TABLE ... DROP CONSTRAINT, and the check
-- constraints the PostgreSQL down drops were never created here, so only the
-- columns and the table are removed.
-- Source: 20261231006670_accounting_app_credentials.tx.down.sql

ALTER TABLE "accounting_connections" DROP COLUMN "app_fingerprint";

--bun:split

ALTER TABLE "accounting_connections" DROP COLUMN "app_environment";

--bun:split

ALTER TABLE "accounting_connections" DROP COLUMN "app_source";

--bun:split

DROP TABLE IF EXISTS "accounting_app_credentials";
