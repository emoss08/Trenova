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
