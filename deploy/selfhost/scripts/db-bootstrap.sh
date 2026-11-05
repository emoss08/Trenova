#!/bin/sh
# One-shot step between the migrations and the services that read the
# database (compose service db-bootstrap). Idempotent; runs on every start.
#
# Scopes the gtc_publication publication to the tables GTC's sink file
# routes. GTC's own auto-create would publish FOR ALL TABLES and backfill
# every one of them, so the deploy owns the publication instead. A table
# added to the sink file later is streamed from then on; backfill it with
# POST /backfill on GTC.
#
# With GTC_ROLE set (the Helm chart, where GTC logs in as a role of its own
# rather than the owner), that role is also given what GTC reads: SELECT on
# the published tables, for its backfill, and membership in
# trenova_rls_bypass, so row-level security does not hide rows from it.
#
# The first organization and administrator are not created here: the
# tms-bootstrap service runs `trenova db bootstrap` for that.
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

if [ -n "${GTC_ROLE:-}" ]; then
  psql -v ON_ERROR_STOP=1 -qt -v role="${GTC_ROLE}" -v tables="${tables}" <<'SQL'
SELECT format('GRANT USAGE ON SCHEMA public TO %I', :'role')
\gexec
SELECT format('GRANT SELECT ON TABLE %s TO %I', :'tables', :'role')
\gexec
SELECT format('GRANT trenova_rls_bypass TO %I', :'role')
WHERE EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'trenova_rls_bypass')
\gexec
SELECT :'role' || ': reads ' || count(*) || ' tables'
FROM information_schema.role_table_grants
WHERE grantee = :'role' AND privilege_type = 'SELECT';
SQL
fi
