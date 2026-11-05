#!/bin/sh
# The roles and databases the migrations and the other services sign in with,
# as a superuser: the first step of the Helm chart's release Job
# (templates/release-job.yaml), and the db-prepare service in compose.yml.
# Idempotent; runs on every install, upgrade and start, and puts each role's
# password back to the one in its Secret or .env, so a restored cluster takes
# the release's passwords again.
#
# Temporal and GTC own what they keep. A database whose objects another role
# made (an install from before they had roles of their own, a restored dump)
# hands them over here, because neither role could migrate or read them.
#
#   OWNER, OWNER_PASSWORD        owns the tables; migrations and seeds run as
#                                it. A superuser: migrations create event
#                                triggers and extensions, and it provisions
#                                the application and system roles. Only the
#                                release Job holds its password.
#   TEMPORAL_ROLE, ..._PASSWORD  owns the temporal and temporal_visibility databases
#   GTC_ROLE, GTC_PASSWORD       GTC's logical replication login. It keeps its
#                                own state (gtc_backfill_state) in a schema it
#                                owns, first on its search path, so it needs no
#                                CREATE on public.
#   PG* (libpq)                  the superuser's connection to the app database
set -eu

: "${OWNER:?}" "${OWNER_PASSWORD:?}" "${TEMPORAL_ROLE:?}" "${TEMPORAL_PASSWORD:?}"
: "${GTC_ROLE:?}" "${GTC_PASSWORD:?}" "${PGDATABASE:?}"

deadline=$(($(date +%s) + ${WAIT_SECONDS:-900}))
until psql -qtAc 'SELECT 1' >/dev/null 2>&1; do
  if [ "$(date +%s)" -ge "${deadline}" ]; then
    echo "PostgreSQL at ${PGHOST} did not accept connections in time" >&2
    psql -qtAc 'SELECT 1'
    exit 1
  fi
  echo "Waiting for PostgreSQL at ${PGHOST}"
  sleep 5
done

psql -v ON_ERROR_STOP=1 -qt \
  -v owner="${OWNER}" -v owner_pw="${OWNER_PASSWORD}" \
  -v temporal="${TEMPORAL_ROLE}" -v temporal_pw="${TEMPORAL_PASSWORD}" \
  -v gtc="${GTC_ROLE}" -v gtc_pw="${GTC_PASSWORD}" \
  -v app_db="${PGDATABASE}" <<'SQL'
SELECT format('ALTER ROLE %I WITH LOGIN SUPERUSER PASSWORD %L', :'owner', :'owner_pw')
\gexec

SELECT format('CREATE ROLE %I', r)
FROM unnest(ARRAY[:'temporal', :'gtc']) AS r
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = r)
\gexec
SELECT format('ALTER ROLE %I WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION PASSWORD %L', :'temporal', :'temporal_pw')
\gexec
SELECT format('ALTER ROLE %I WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE REPLICATION PASSWORD %L', :'gtc', :'gtc_pw')
\gexec

SELECT format('CREATE DATABASE %I OWNER %I', d, :'temporal')
FROM unnest(ARRAY['temporal', 'temporal_visibility']) AS d
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = d)
\gexec
SELECT format('ALTER DATABASE %I OWNER TO %I', d, :'temporal')
FROM unnest(ARRAY['temporal', 'temporal_visibility']) AS d
\gexec

SELECT format('GRANT CONNECT ON DATABASE %I TO %I', :'app_db', :'gtc')
\gexec
SELECT format('CREATE SCHEMA IF NOT EXISTS gtc AUTHORIZATION %I', :'gtc')
\gexec
SELECT format('ALTER ROLE %I IN DATABASE %I SET search_path = gtc, public', :'gtc', :'app_db')
\gexec
SELECT 'ALTER TABLE public.gtc_backfill_state SET SCHEMA gtc'
WHERE to_regclass('public.gtc_backfill_state') IS NOT NULL
  AND to_regclass('gtc.gtc_backfill_state') IS NULL
\gexec
SELECT format('ALTER TABLE gtc.gtc_backfill_state OWNER TO %I', :'gtc')
WHERE to_regclass('gtc.gtc_backfill_state') IS NOT NULL
\gexec
SQL

for database in temporal temporal_visibility; do
  psql -v ON_ERROR_STOP=1 -qt -d "${database}" -v temporal="${TEMPORAL_ROLE}" <<'SQL'
SELECT format('ALTER TABLE public.%I OWNER TO %I', c.relname, :'temporal')
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
JOIN pg_roles r ON r.oid = c.relowner
WHERE n.nspname = 'public'
  AND c.relkind IN ('r', 'p', 'v', 'm')
  AND r.rolname <> :'temporal'
  AND NOT EXISTS (
    SELECT 1 FROM pg_depend d
    WHERE d.classid = 'pg_class'::regclass AND d.objid = c.oid AND d.deptype = 'e'
  )
\gexec
SELECT format(
  'ALTER ROUTINE public.%I(%s) OWNER TO %I',
  p.proname, pg_get_function_identity_arguments(p.oid), :'temporal'
)
FROM pg_proc p
JOIN pg_namespace n ON n.oid = p.pronamespace
JOIN pg_roles r ON r.oid = p.proowner
WHERE n.nspname = 'public'
  AND r.rolname <> :'temporal'
  AND NOT EXISTS (
    SELECT 1 FROM pg_depend d
    WHERE d.classid = 'pg_proc'::regclass AND d.objid = p.oid AND d.deptype = 'e'
  )
\gexec
SQL
done

echo "PostgreSQL is prepared: roles ${OWNER}, ${TEMPORAL_ROLE} and ${GTC_ROLE}"
