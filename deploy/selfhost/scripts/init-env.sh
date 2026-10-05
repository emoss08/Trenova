#!/usr/bin/env sh
# Creates .env for a new install: copies .env.example, generates every secret
# marked [generated], and, when given a hostname, sets DOMAIN to it and
# STORAGE_DOMAIN to storage.<hostname>. It also asks for the first
# organization and administrator, and generates the administrator's password
# when none is entered.
#
#   ./scripts/init-env.sh                       # trial on https://localhost
#   ./scripts/init-env.sh trenova.example.com   # public install
#
# Without a terminal, give the administrator in the environment instead:
# TRENOVA_BOOTSTRAP_ORG_NAME, TRENOVA_BOOTSTRAP_ADMIN_NAME and
# TRENOVA_BOOTSTRAP_ADMIN_EMAIL, and optionally TRENOVA_BOOTSTRAP_ADMIN_PASSWORD.
#
# Refuses to overwrite an existing .env: regenerating secrets on a running
# install locks it out of its own data.
set -eu

cd "$(dirname "$0")/.."

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  sed -n '2,17p' "$0" | sed 's/^# \{0,1\}//'
  exit 0
fi

domain="${1:-}"
if [ -n "${domain}" ] && ! printf '%s' "${domain}" | grep -Eq '^[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?)+$'; then
  echo "'${domain}' is not a hostname (give it without https:// or a path)." >&2
  exit 1
fi

if [ -e .env ]; then
  echo ".env already exists; refusing to overwrite it." >&2
  exit 1
fi

if ! command -v openssl >/dev/null 2>&1; then
  echo "openssl is required to generate secrets." >&2
  exit 1
fi

min_password_length=12
interactive=false
if [ -t 0 ] && [ -t 2 ]; then
  interactive=true
fi

secret() {
  openssl rand -hex 32
}

generate_password() {
  openssl rand -base64 32 | tr -d '/+=\n' | cut -c1-24
}

ask() {
  label="$1"
  answer="$2"
  if [ -z "${answer}" ] && [ "${interactive}" = true ]; then
    printf '%s: ' "${label}" >&2
    IFS= read -r answer || answer=""
  fi
  printf '%s' "${answer}"
}

ask_secret() {
  label="$1"
  printf '%s: ' "${label}" >&2
  trap 'stty echo 2>/dev/null' EXIT INT TERM
  stty -echo
  IFS= read -r answer || answer=""
  stty echo
  trap - EXIT INT TERM
  printf '\n' >&2
  printf '%s' "${answer}"
}

require_value() {
  name="$1"
  value="$2"
  if [ -z "${value}" ]; then
    echo "${name} is required. Run this in a terminal to be asked, or set it in the environment." >&2
    exit 1
  fi
  case "${value}" in
    *"'"*)
      echo "${name} cannot contain a single quote (')." >&2
      exit 1
      ;;
  esac
  if [ "$(printf '%s' "${value}" | wc -l)" -ne 0 ]; then
    echo "${name} must be a single line." >&2
    exit 1
  fi
}

if [ "${interactive}" = true ]; then
  echo "The first organization and its administrator (change both later in the app):" >&2
fi
org_name="$(ask 'Organization name' "${TRENOVA_BOOTSTRAP_ORG_NAME:-}")"
require_value TRENOVA_BOOTSTRAP_ORG_NAME "${org_name}"
admin_name="$(ask 'Administrator full name' "${TRENOVA_BOOTSTRAP_ADMIN_NAME:-}")"
require_value TRENOVA_BOOTSTRAP_ADMIN_NAME "${admin_name}"
admin_email="$(ask 'Administrator email (signs in with it)' "${TRENOVA_BOOTSTRAP_ADMIN_EMAIL:-}")"
require_value TRENOVA_BOOTSTRAP_ADMIN_EMAIL "${admin_email}"
if ! printf '%s' "${admin_email}" | grep -Eq '^[^@[:space:]]+@[^@[:space:]]+\.[^@[:space:]]+$'; then
  echo "'${admin_email}' is not an email address." >&2
  exit 1
fi

admin_password="${TRENOVA_BOOTSTRAP_ADMIN_PASSWORD:-}"
if [ -z "${admin_password}" ] && [ "${interactive}" = true ]; then
  admin_password="$(ask_secret "Administrator password (at least ${min_password_length} characters; empty generates one)")"
  if [ -n "${admin_password}" ]; then
    confirmation="$(ask_secret 'Repeat the password')"
    if [ "${admin_password}" != "${confirmation}" ]; then
      echo "The passwords did not match." >&2
      exit 1
    fi
  fi
fi

generated_password=false
if [ -z "${admin_password}" ]; then
  admin_password="$(generate_password)"
  generated_password=true
fi
require_value TRENOVA_BOOTSTRAP_ADMIN_PASSWORD "${admin_password}"
if [ "${#admin_password}" -lt "${min_password_length}" ]; then
  echo "The administrator password needs at least ${min_password_length} characters." >&2
  exit 1
fi

set_value() {
  tmp="$(mktemp .env.XXXXXX)"
  SET_NAME="$1" SET_VALUE="$2" awk '
    index($0, ENVIRON["SET_NAME"] "=") == 1 { print ENVIRON["SET_NAME"] "=" ENVIRON["SET_VALUE"]; next }
    { print }
  ' .env > "${tmp}"
  cat "${tmp}" > .env
  rm -f "${tmp}"
}

set_quoted() {
  set_value "$1" "'$2'"
}

umask 077
cp .env.example .env

for name in \
  TRENOVA_DATABASE_PASSWORD \
  TRENOVA_REDIS_PASSWORD \
  TRENOVA_MEILI_MASTER_KEY \
  TRENOVA_STORAGE_SECRETKEY \
  TRENOVA_SECURITY_SESSION_SECRET \
  TRENOVA_SECURITY_ENCRYPTION_KEY \
  TRENOVA_SYSTEM_SYSTEMUSERPASSWORD; do
  set_value "${name}" "$(secret)"
done

audit_key_id="selfhost-$(date -u +%Y%m%d)"
set_value TRENOVA_AI_AUDIT_CHAIN_KEYS "${audit_key_id}:$(secret)"
set_value TRENOVA_AI_AUDIT_CHAIN_ACTIVE_KEY_ID "${audit_key_id}"

if [ -n "${domain}" ]; then
  set_value DOMAIN "${domain}"
  set_value STORAGE_DOMAIN "storage.${domain}"
fi

set_quoted TRENOVA_BOOTSTRAP_ORG_NAME "${org_name}"
set_quoted TRENOVA_BOOTSTRAP_ADMIN_NAME "${admin_name}"
set_quoted TRENOVA_BOOTSTRAP_ADMIN_EMAIL "${admin_email}"
set_quoted TRENOVA_BOOTSTRAP_ADMIN_PASSWORD "${admin_password}"

echo "Created .env (mode 600) with generated secrets."
echo
grep -E '^(DOMAIN|STORAGE_DOMAIN|TRENOVA_VERSION)=' .env
echo
echo "First administrator: ${admin_email}"
if [ "${generated_password}" = true ]; then
  echo "Generated password:  ${admin_password}"
  echo "It is shown only this once; it is also in .env as TRENOVA_BOOTSTRAP_ADMIN_PASSWORD"
  echo "until you delete it there, which you may do once you have signed in."
fi
echo
echo "Back up .env now, apart from this host. TRENOVA_SECURITY_ENCRYPTION_KEY"
echo "cannot be recovered, and encrypted data cannot be read without it."
echo "Then: docker compose up -d"
