#!/bin/sh
# One-shot step between the migrations and the services that read the
# database (compose service db-bootstrap). Idempotent; runs on every start.
#
# 1. Scopes the gtc_publication publication to the tables GTC's sink file
#    routes. GTC's own auto-create would publish FOR ALL TABLES and backfill
#    every one of them, so the deploy owns the publication instead. A table
#    added to the sink file later is streamed from then on; backfill it with
#    POST /backfill on GTC.
#
# 2. Once per database, after the base seeds have created the administrator:
#    the seeded passwords are public, so `admin` must choose a new password at
#    its first sign-in and the two seeded organization administrators
#    (admin-logistics, admin-transport) are locked until an administrator
#    sets them a password. A database setting records that this was done, so
#    a password chosen later is never touched again, and only an
#    administrator seeded within the last day qualifies, so restoring an
#    existing database into a new volume never triggers it.
set -eu

SINKS_FILE="${SINKS_FILE:-/config/sinks.yaml}"
PUBLICATION="gtc_publication"

tables="$(grep -oE '^[[:space:]]+public\.[a-z_]+:' "${SINKS_FILE}" | tr -d ' :' | sort -u | tr '\n' ',' | sed 's/,$//')"
if [ -z "${tables}" ]; then
  echo "No public.<table> entries found in ${SINKS_FILE}" >&2
  exit 1
fi

psql -v ON_ERROR_STOP=1 -qt -v pub="${PUBLICATION}" -v tables="${tables}" <<'SQL'
SELECT CASE
  WHEN EXISTS (SELECT 1 FROM pg_publication WHERE pubname = :'pub')
    THEN format('ALTER PUBLICATION %I SET TABLE %s', :'pub', :'tables')
  ELSE format('CREATE PUBLICATION %I FOR TABLE %s', :'pub', :'tables')
END
\gexec
SELECT 'gtc_publication: ' || count(*) || ' tables'
FROM pg_publication_tables WHERE pubname = :'pub';
SQL

psql -v ON_ERROR_STOP=1 -qt <<'SQL'
SELECT
  current_setting('trenova_selfhost.seed_accounts_secured', true) IS DISTINCT FROM 'on'
  AND EXISTS (
    SELECT 1 FROM users
    WHERE username = 'admin'
      AND created_at > extract(epoch FROM now())::bigint - 86400
  ) AS pending
\gset
\if :pending
BEGIN;
UPDATE users SET must_change_password = true WHERE username = 'admin';
UPDATE users SET password = '!locked'
WHERE username IN ('admin-logistics', 'admin-transport');
SELECT format('ALTER DATABASE %I SET trenova_selfhost.seed_accounts_secured = %L', current_database(), 'on')
\gexec
COMMIT;
SELECT 'Seeded accounts secured: admin must change its password at first sign-in; admin-logistics and admin-transport are locked';
\else
SELECT 'Seeded accounts: nothing to do';
\endif
SQL
