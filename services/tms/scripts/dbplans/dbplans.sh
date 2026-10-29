#!/usr/bin/env bash
# Plan guard: explain every statement shape the integration suite recorded
# (KEEL_SHAPES) against a database with volume data and fresh statistics, and
# fail on a sequential scan of a large table or a page sorted without an index.
# Run from services/tms after the suite; `task db-plans` runs both, and CI runs
# this after its integration step. See "Plan guard" in
# docs/engineering/backend/data-access.md.
#
# Environment:
#   KEEL_SHAPES          the shapes file the suite wrote (required)
#   PLANS_CONTAINER      the Docker container of the PostgreSQL 16+ server (required)
#   PLANS_DATABASE       the database to build and plan against (trenova_plans)
#   PLANS_SHIPMENTS      shipments of volume data to generate (20000)
#   TRENOVA_DATABASE_*   host, port, user, password and sslMode of that server, as
#                        trenova-db reads them; TRENOVA_DATABASE_NAME is set here
set -euo pipefail

shapes="${KEEL_SHAPES:?KEEL_SHAPES must name the shapes file the integration suite wrote}"
container="${PLANS_CONTAINER:?PLANS_CONTAINER must name the PostgreSQL container}"
database="${PLANS_DATABASE:-trenova_plans}"
shipments="${PLANS_SHIPMENTS:-20000}"

if [[ ! "$database" =~ ^[a-z_][a-z0-9_]*$ ]]; then
  echo "PLANS_DATABASE must be a plain lower-case identifier" >&2
  exit 2
fi
if [ ! -s "$shapes" ]; then
  echo "no statement shapes in $shapes: run the integration suite with KEEL_SHAPES set" >&2
  exit 2
fi

run_psql() {
  docker exec -i "$container" psql -v ON_ERROR_STOP=1 -X -q -U "${TRENOVA_DATABASE_USER:-postgres}" "$@"
}

run_psql -d postgres -c "DROP DATABASE IF EXISTS $database WITH (FORCE)"
run_psql -d postgres -c "CREATE DATABASE $database"

export TRENOVA_DATABASE_NAME="$database"
export APP_ENV="${APP_ENV:-test}"

go run ./cmd/devdb up -env test
go run ./cmd/devdb volume -shipments "$shipments" -months 12 -seed 1
run_psql -d "$database" -c "VACUUM (ANALYZE)"

printf 'planning the shapes of %s recorded lines\n' "$(wc -l <"$shapes" | tr -d ' ')"
go run ./cmd/db plans -plain -shapes "$shapes" -sort-limit
