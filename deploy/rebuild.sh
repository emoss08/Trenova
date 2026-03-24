#!/bin/bash
set -e

cd "$(dirname "$0")"

source .env

SERVICES="${@:-tms-api tms-worker caddy}"

echo "Rebuilding: $SERVICES"
docker compose build $SERVICES --no-cache

echo "Restarting services..."
docker compose up -d $SERVICES

echo "Waiting for health checks..."
sleep 5
docker compose ps

echo "Done!"
