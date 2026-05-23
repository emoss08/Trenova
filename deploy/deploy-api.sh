#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.api.yml"

usage() {
  cat <<'USAGE'
Usage:
  ./deploy-api.sh [version]

Examples:
  ./deploy-api.sh 0.0.6
  TRENOVA_VERSION=latest ./deploy-api.sh

Environment:
  SKIP_GIT_PULL=true   Do not pull the latest master before deploying.
  TRENOVA_VERSION      Image tag to deploy when [version] is not provided.
  TRENOVA_POSTGRES_VERSION
                       Postgres image tag. Defaults to latest because Postgres
                       is released independently from the TMS image.
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

VERSION="${1:-${TRENOVA_VERSION:-latest}}"

if [[ ! -f "${SCRIPT_DIR}/.env" ]]; then
  echo "Missing ${SCRIPT_DIR}/.env"
  echo "Copy .env.example to .env and configure secrets before deploying."
  exit 1
fi

if [[ "${SKIP_GIT_PULL:-false}" != "true" && -d "${REPO_DIR}/.git" ]]; then
  echo "Pulling latest master..."
  git -C "${REPO_DIR}" pull --ff-only origin master
fi

export TRENOVA_VERSION="${VERSION}"
export TRENOVA_POSTGRES_VERSION="${TRENOVA_POSTGRES_VERSION:-latest}"

echo "Deploying Trenova API stack with image tag: ${TRENOVA_VERSION}"
echo "Using Postgres image tag: ${TRENOVA_POSTGRES_VERSION}"
echo "Using compose file: ${COMPOSE_FILE}"

cd "${SCRIPT_DIR}"

echo "Pulling images..."
docker compose -f "${COMPOSE_FILE}" pull

echo "Starting infrastructure services..."
docker compose -f "${COMPOSE_FILE}" up -d \
  postgres \
  redis \
  minio \
  meilisearch \
  temporal \
  pgbouncer

echo "Running migrations..."
docker compose -f "${COMPOSE_FILE}" up \
  --force-recreate \
  --no-deps \
  tms-migrate

echo "Starting application services..."
docker compose -f "${COMPOSE_FILE}" up -d \
  --force-recreate \
  tms-api \
  tms-worker \
  cloudflared

echo "Current service status:"
docker compose -f "${COMPOSE_FILE}" ps

echo
echo "Recent API logs:"
docker compose -f "${COMPOSE_FILE}" logs --tail=40 tms-api

echo
echo "Deployment complete."
