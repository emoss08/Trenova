#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
SPEC_FILE="${ROOT_DIR}/internal/openapi/samsara-api-2025-10-23.json"
OUT_FILE="${ROOT_DIR}/internal/samsaraspec/types.gen.go"

if [[ ! -f "${SPEC_FILE}" ]]; then
  echo "spec file not found: ${SPEC_FILE}" >&2
  exit 1
fi

go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.6.0 \
  -generate types \
  -package samsaraspec \
  "${SPEC_FILE}" > "${OUT_FILE}"

gofmt -w "${OUT_FILE}"
