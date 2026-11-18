#!/usr/bin/env bash
# Checks lib.sh against this image's PostgreSQL, as the image is built (the
# Dockerfile runs it as postgres): a copy of a small database restored the
# way export.sh restores a backup, without hot standby, takes connections
# only once wait_for_promotion says so, and a server that stops or never
# finishes is reported as such.
set -euo pipefail
. "$(dirname "$0")/lib.sh"
export WAIT_INTERVAL=1

t="$(mktemp -d)"
trap 'for d in "${t}"/*/; do pg_ctl -D "${d}" -m immediate stop >/dev/null 2>&1 || true; done; rm -rf "${t}"' EXIT

check() {
  if ! eval "$2"; then
    echo "test.sh: $1" >&2
    exit 1
  fi
}

initdb -D "${t}/primary" -U postgres --auth=trust >/dev/null
mkdir -p "${t}/s0" "${t}/s1" "${t}/s2"
pg_ctl -D "${t}/primary" -o "-c listen_addresses='' -c unix_socket_directories='${t}/s0'" -w start >/dev/null
psql -h "${t}/s0" -U postgres -d postgres -XAtqc "CREATE TABLE kept AS SELECT generate_series(1, 1000) AS n"
pg_basebackup -h "${t}/s0" -U postgres -D "${t}/base" -X stream
pg_ctl -D "${t}/primary" -w stop >/dev/null

# A restored copy, configured as export.sh configures one; its restore
# command takes a while and finds nothing, so recovery lasts a few seconds
# and then promotes.
restored() {
  cp -a "${t}/base" "${t}/$1"
  cat >"${t}/$1/postgresql.auto.conf" <<CONF
listen_addresses = ''
unix_socket_directories = '${t}/$2'
hot_standby = off
archive_mode = off
restore_command = 'sleep $3; false'
recovery_target_timeline = 'latest'
CONF
  touch "${t}/$1/recovery.signal"
  # Started without waiting: a restore command this slow can keep pg_ctl -w
  # waiting before recovery begins. Only the postmaster has to be up.
  pg_ctl -D "${t}/$1" -l "${t}/$1.log" -W start >/dev/null
  for _ in $(seq 1 60); do
    if pg_ctl -D "${t}/$1" status >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  echo "test.sh: the restored copy $1 did not start" >&2
  exit 1
}

# export.sh's own read-only names, which the function must not touch.
readonly pgdata="${t}/none" socket="${t}/none"

restored replayed s1 3
waited=0
wait_for_promotion "${t}/replayed" "${t}/s1" 60 2>"${t}/stderr" || waited=$?
check "a server that promotes is waited for (got ${waited})" '[ "${waited}" = 0 ]'
check "waiting says nothing: $(cat "${t}/stderr")" '[ ! -s "${t}/stderr" ]'
check "the server has left recovery" \
  '[ "$(psql -h "${t}/s1" -U postgres -d postgres -XAtc "SELECT pg_is_in_recovery()")" = f ]'
check "the restored copy holds every row" \
  '[ "$(psql -h "${t}/s1" -U postgres -d postgres -XAtc "SELECT count(*) FROM kept")" = 1000 ]'
pg_ctl -D "${t}/replayed" -w stop >/dev/null
waited=0
wait_for_promotion "${t}/replayed" "${t}/s1" 60 || waited=$?
check "a stopped server is reported (got ${waited})" '[ "${waited}" = 1 ]'

restored slow s2 60
waited=0
wait_for_promotion "${t}/slow" "${t}/s2" 2 || waited=$?
check "a server still in recovery at the deadline is reported (got ${waited})" '[ "${waited}" = 2 ]'

echo "test.sh: the runner's recovery wait works"
