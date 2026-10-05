#!/usr/bin/env sh
# Creates .env for a new install: copies .env.example, generates every secret
# marked [generated], and, when given a hostname, sets DOMAIN to it and
# STORAGE_DOMAIN to storage.<hostname>.
#
#   ./scripts/init-env.sh                       # trial on https://localhost
#   ./scripts/init-env.sh trenova.example.com   # public install
#
# Refuses to overwrite an existing .env: regenerating secrets on a running
# install locks it out of its own data.
set -eu

cd "$(dirname "$0")/.."

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  sed -n '2,11p' "$0" | sed 's/^# \{0,1\}//'
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

secret() {
  openssl rand -hex 32
}

set_value() {
  name="$1"
  value="$2"
  tmp="$(mktemp .env.XXXXXX)"
  awk -v name="${name}" -v value="${value}" '
    index($0, name "=") == 1 { print name "=" value; next }
    { print }
  ' .env > "${tmp}"
  cat "${tmp}" > .env
  rm -f "${tmp}"
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

echo "Created .env (mode 600) with generated secrets."
echo
grep -E '^(DOMAIN|STORAGE_DOMAIN|TRENOVA_VERSION)=' .env
echo
echo "Back up .env now, apart from this host. TRENOVA_SECURITY_ENCRYPTION_KEY"
echo "cannot be recovered, and encrypted data cannot be read without it."
echo "Then: docker compose up -d"
