# The export runner's functions that can be tried without object storage:
# export.sh sources this file, and test.sh, run when the image is built,
# checks them against a real PostgreSQL.

# wait_for_promotion waits until the PostgreSQL at data directory $1, whose
# socket is in $2, has finished recovery and takes connections, for at most
# $3 seconds. pg_ctl -w cannot be trusted for this: without hot standby, it
# reports a server in recovery as started (PostgreSQL marks it a ready
# standby as soon as recovery begins), while every connection is refused
# until the server has replayed every change and promoted itself. It
# returns 1 when the server stopped first, and 2 when the time ran out.
wait_for_promotion() {
  # Named apart from export.sh's own (read-only) variables.
  local wait_dir="$1" wait_socket="$2" wait_until=$((SECONDS + $3))
  until [ "$(psql -h "${wait_socket}" -U postgres -d postgres -XAtc "SELECT pg_is_in_recovery()" 2>/dev/null)" = "f" ]; do
    if ! pg_ctl -D "${wait_dir}" status >/dev/null 2>&1; then
      return 1
    fi
    if [ "${SECONDS}" -ge "${wait_until}" ]; then
      return 2
    fi
    sleep "${WAIT_INTERVAL:-5}"
  done
}
