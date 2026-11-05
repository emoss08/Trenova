#!/bin/sh
# First step of the release Job (templates/release-job.yaml), as the
# CloudNativePG superuser: the roles and databases the migrations and the
# other services sign in with. Idempotent; runs on every install and upgrade,
# and puts each role's password back to the one in its Secret, so a restored
# cluster takes the release's passwords again.
#
#   OWNER, OWNER_PASSWORD        owns the tables; migrations and seeds run as
#                                it. A superuser: migrations create event
#                                triggers and extensions, and it provisions
#                                the application and system roles. Only the
#                                release Job holds its password.
#   TEMPORAL_ROLE, ..._PASSWORD  owns the temporal and temporal_visibility databases
#   GTC_ROLE, GTC_PASSWORD       GTC's logical replication login
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
SQL

echo "PostgreSQL is prepared: roles ${OWNER}, ${TEMPORAL_ROLE} and ${GTC_ROLE}"
