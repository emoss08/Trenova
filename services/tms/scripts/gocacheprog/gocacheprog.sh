#!/usr/bin/env bash
# GOCACHEPROG helper: runs gobuildcache against the shared Go build cache on
# Cloudflare R2. The go command passes its environment to every test binary,
# so the store's endpoint and keys live only in this process: they come from a
# credentials file (GO_REMOTE_CACHE_CREDENTIALS, default
# ~/.config/trenova/go-remote-cache.env) or, in a cloud session, from the
# environment's R2_* variables, and every AWS_* variable inherited from the
# caller is dropped before gobuildcache starts.
#
# The cache is read-only unless the credentials file itself sets
# GO_REMOTE_CACHE_WRITE=true, which only CI on pushes to master does. Without
# credentials it runs as a local disk cache, so a build never fails for lack
# of them.
#
# `gocacheprog.sh clear-local` empties the local directory, which gobuildcache
# never trims. See docs/engineering/go-build-cache.md.
set -euo pipefail

cache_root="${GOBUILDCACHE_CACHE_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/trenova/gobuildcache}"
lock_dir="${GOBUILDCACHE_LOCK_DIR:-$cache_root.locks}"

bin="${GOBUILDCACHE_BIN:-}"
if [ -z "$bin" ]; then
	for candidate in "$(command -v gobuildcache || true)" \
		"${GOBIN:+$GOBIN/gobuildcache}" \
		"${GOPATH:-$HOME/go}/bin/gobuildcache"; do
		if [ -n "$candidate" ] && [ -x "$candidate" ]; then
			bin="$candidate"
			break
		fi
	done
fi
if [ -z "$bin" ]; then
	echo "gocacheprog: gobuildcache not found; run 'task go-remote-cache-install' in services/tms" >&2
	exit 1
fi

endpoint=""
access_key_id=""
secret_access_key=""
bucket="trenova-go-build-cache"
write="false"

credentials="${GO_REMOTE_CACHE_CREDENTIALS:-${XDG_CONFIG_HOME:-$HOME/.config}/trenova/go-remote-cache.env}"
if [ -f "$credentials" ]; then
	while IFS='=' read -r key value || [ -n "$key" ]; do
		case "$key" in
		R2_BUILD_CACHE_ENDPOINT) endpoint="$value" ;;
		R2_BUILD_CACHE_ACCESS_KEY_ID) access_key_id="$value" ;;
		R2_BUILD_CACHE_SECRET_ACCESS_KEY) secret_access_key="$value" ;;
		R2_BUILD_CACHE_BUCKET) bucket="$value" ;;
		GO_REMOTE_CACHE_WRITE) write="$value" ;;
		esac
	done <"$credentials"
elif [ -n "${R2_ACCESS_KEY_ID:-}" ] && [ -n "${R2_SECRET_ACCESS_KEY:-}" ] && [ -n "${R2_S3_API_ENDPOINT:-}" ]; then
	endpoint="$R2_S3_API_ENDPOINT"
	access_key_id="$R2_ACCESS_KEY_ID"
	secret_access_key="$R2_SECRET_ACCESS_KEY"
fi

while IFS= read -r name; do
	case "$name" in
	AWS_* | R2_* | GOBUILDCACHE_* | BACKEND | BACKEND_TYPE | S3_* | READ_ONLY | CACHE_DIR | LOCK_DIR | COMPRESSION | ASYNC_BACKEND | ERROR_RATE)
		unset "$name"
		;;
	esac
done < <(compgen -e)

export GOBUILDCACHE_CACHE_DIR="$cache_root"
export GOBUILDCACHE_LOCK_DIR="$lock_dir"
export GOBUILDCACHE_PRINT_STATS="${GO_REMOTE_CACHE_STATS:-false}"

if [ "${1:-}" = clear-local ]; then
	exec "$bin" clear-local
fi

if [ -z "$endpoint" ] || [ -z "$access_key_id" ] || [ -z "$secret_access_key" ]; then
	echo "gocacheprog: no credentials in $credentials; using the local cache only" >&2
	export GOBUILDCACHE_BACKEND_TYPE=disk
	exec "$bin"
fi

export GOBUILDCACHE_BACKEND_TYPE=s3
export GOBUILDCACHE_S3_BUCKET="$bucket"
export GOBUILDCACHE_AWS_ACCESS_KEY_ID="$access_key_id"
export GOBUILDCACHE_AWS_SECRET_ACCESS_KEY="$secret_access_key"
export GOBUILDCACHE_AWS_REGION=auto
export GOBUILDCACHE_AWS_S3_USE_PATH_STYLE=true
export GOBUILDCACHE_READ_ONLY=true
if [ "$write" = true ]; then
	export GOBUILDCACHE_READ_ONLY=false
fi
export AWS_ENDPOINT_URL_S3="$endpoint"
export AWS_CONFIG_FILE=/dev/null
export AWS_SHARED_CREDENTIALS_FILE=/dev/null
export AWS_EC2_METADATA_DISABLED=true
exec "$bin"
