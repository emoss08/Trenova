#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${GREEN}[r2-cors]${NC} $1"; }
warn() { echo -e "${YELLOW}[r2-cors]${NC} $1"; }
info() { echo -e "${BLUE}[r2-cors]${NC} $1"; }
err() { echo -e "${RED}[r2-cors]${NC} $1" >&2; }

usage() {
    cat <<'EOF'
Usage: scripts/configure-r2-cors.sh [options]

Configures Cloudflare R2 bucket CORS for browser-based presigned uploads.

Required:
  CLOUDFLARE_API_TOKEN       Cloudflare API token with R2 bucket CORS edit access.
  TRENOVA_STORAGE_BUCKET     R2 bucket name.

Required unless derivable from TRENOVA_STORAGE_ENDPOINT:
  CLOUDFLARE_ACCOUNT_ID      Cloudflare account ID.

Options:
  --env-file PATH            Load variables from an env file before running.
                             Defaults to .env when present.
  --origin ORIGIN            Allowed browser origin. Can be repeated.
                             Defaults to http://localhost:5173.
  --dry-run                  Print the request payload without calling Cloudflare.
  -h, --help                 Show this help text.

Optional env vars:
  R2_CORS_ORIGINS            Comma-separated origins. Ignored when --origin is used.
  R2_CORS_METHODS            Comma-separated methods. Default: PUT,GET,HEAD.
  R2_CORS_HEADERS            Comma-separated allowed headers. Default: Content-Type.
  R2_CORS_EXPOSE_HEADERS     Comma-separated exposed headers. Default: ETag.
  R2_CORS_MAX_AGE_SECONDS    Preflight cache seconds. Default: 3600.

Examples:
  CLOUDFLARE_API_TOKEN=... TRENOVA_STORAGE_BUCKET=trenova-test \
    TRENOVA_STORAGE_ENDPOINT=https://<account-id>.r2.cloudflarestorage.com \
    scripts/configure-r2-cors.sh

  scripts/configure-r2-cors.sh --origin http://localhost:5173 --origin http://127.0.0.1:5173
EOF
}

require_command() {
    if ! command -v "$1" >/dev/null 2>&1; then
        err "$1 is required"
        exit 1
    fi
}

python_cmd=()

detect_python() {
    if command -v python3 >/dev/null 2>&1; then
        python_cmd=(python3)
        return
    fi

    if command -v python >/dev/null 2>&1; then
        python_cmd=(python)
        return
    fi

    if command -v uv >/dev/null 2>&1; then
        python_cmd=(uv run --quiet python)
        return
    fi

    err "python3, python, or uv is required"
    exit 1
}

run_python() {
    "${python_cmd[@]}" "$@"
}

load_env_file() {
    local env_file=$1

    if [ -z "$env_file" ]; then
        return
    fi

    if [ ! -f "$env_file" ]; then
        err "Env file not found: ${env_file}"
        exit 1
    fi

    set -a
    # shellcheck disable=SC1090
    source "$env_file"
    set +a
}

trim() {
    local value=$1
    value="${value#"${value%%[![:space:]]*}"}"
    value="${value%"${value##*[![:space:]]}"}"
    printf '%s' "$value"
}

append_csv_items() {
    local csv=$1
    local -n out_ref=$2
    local item

    IFS=',' read -ra items <<< "$csv"
    for item in "${items[@]}"; do
        item="$(trim "$item")"
        if [ -n "$item" ]; then
            out_ref+=("$item")
        fi
    done
}

derive_account_id() {
    if [ -n "${CLOUDFLARE_ACCOUNT_ID:-}" ]; then
        printf '%s' "$CLOUDFLARE_ACCOUNT_ID"
        return
    fi

    local endpoint=${TRENOVA_STORAGE_ENDPOINT:-}
    endpoint="${endpoint#http://}"
    endpoint="${endpoint#https://}"
    endpoint="${endpoint%%/*}"

    if [[ "$endpoint" =~ ^([a-zA-Z0-9]+)\.r2\.cloudflarestorage\.com$ ]]; then
        printf '%s' "${BASH_REMATCH[1]}"
        return
    fi

    printf ''
}

json_array() {
    run_python - "$@" <<'PY'
import json
import sys

print(json.dumps(sys.argv[1:], separators=(",", ":")))
PY
}

build_payload() {
    local origins_json methods_json headers_json expose_headers_json
    origins_json="$(json_array "${origins[@]}")"
    methods_json="$(json_array "${methods[@]}")"
    headers_json="$(json_array "${headers[@]}")"
    expose_headers_json="$(json_array "${expose_headers[@]}")"

    cat <<EOF
{
  "rules": [
    {
      "id": "trenova-local-presigned-upload",
      "allowed": {
        "origins": ${origins_json},
        "methods": ${methods_json},
        "headers": ${headers_json}
      },
      "exposeHeaders": ${expose_headers_json},
      "maxAgeSeconds": ${max_age_seconds}
    }
  ]
}
EOF
}

env_file=""
dry_run=false
origins=()

if [ -f .env ]; then
    env_file=".env"
fi

while [[ $# -gt 0 ]]; do
    case "$1" in
        --env-file)
            env_file="${2:-}"
            if [ -z "$env_file" ]; then
                err "--env-file requires a path"
                exit 1
            fi
            shift 2
            ;;
        --origin)
            if [ -z "${2:-}" ]; then
                err "--origin requires a value"
                exit 1
            fi
            origins+=("$2")
            shift 2
            ;;
        --dry-run)
            dry_run=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            err "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

load_env_file "$env_file"
require_command curl
detect_python

if [ ${#origins[@]} -eq 0 ]; then
    if [ -n "${R2_CORS_ORIGINS:-}" ]; then
        append_csv_items "$R2_CORS_ORIGINS" origins
    else
        origins=("http://localhost:5173")
    fi
fi

methods=()
append_csv_items "${R2_CORS_METHODS:-PUT,GET,HEAD}" methods

headers=()
append_csv_items "${R2_CORS_HEADERS:-Content-Type}" headers

expose_headers=()
append_csv_items "${R2_CORS_EXPOSE_HEADERS:-ETag}" expose_headers

max_age_seconds="${R2_CORS_MAX_AGE_SECONDS:-3600}"
if ! [[ "$max_age_seconds" =~ ^[0-9]+$ ]]; then
    err "R2_CORS_MAX_AGE_SECONDS must be an integer"
    exit 1
fi

account_id="$(derive_account_id)"
bucket="${TRENOVA_STORAGE_BUCKET:-}"
api_token="${CLOUDFLARE_API_TOKEN:-}"

if [ -z "$api_token" ]; then
    err "CLOUDFLARE_API_TOKEN is required"
    exit 1
fi

if [ -z "$account_id" ]; then
    err "CLOUDFLARE_ACCOUNT_ID is required when TRENOVA_STORAGE_ENDPOINT is not an R2 endpoint"
    exit 1
fi

if [ -z "$bucket" ]; then
    err "TRENOVA_STORAGE_BUCKET is required"
    exit 1
fi

payload="$(build_payload)"
url="https://api.cloudflare.com/client/v4/accounts/${account_id}/r2/buckets/${bucket}/cors"

info "Bucket: ${bucket}"
info "Account: ${account_id}"
info "Origins: ${origins[*]}"
info "Methods: ${methods[*]}"

if [ "$dry_run" = true ]; then
    warn "Dry run: not calling Cloudflare"
    printf '%s\n' "$payload"
    exit 0
fi

response="$(curl --silent --show-error --fail-with-body \
    --request PUT "$url" \
    --header "Authorization: Bearer ${api_token}" \
    --header "Content-Type: application/json" \
    --data "$payload")"

success="$(run_python - "$response" <<'PY'
import json
import sys

try:
    print("true" if json.loads(sys.argv[1]).get("success") is True else "false")
except Exception:
    print("false")
PY
)"

if [ "$success" != "true" ]; then
    err "Cloudflare API did not report success"
    printf '%s\n' "$response" >&2
    exit 1
fi

log "R2 CORS policy updated"
