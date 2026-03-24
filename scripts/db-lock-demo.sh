#!/usr/bin/env bash

set -euo pipefail

STATE_DIR="${TMPDIR:-/tmp}/trenova-db-lock-demo"
LOCK_KEY="424242"
CONTAINER_NAME="${DB_CONTAINER_NAME:-db}"
DB_NAME="${DB_NAME:-trenova_go_db}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"

mkdir -p "$STATE_DIR"

run_psql() {
  docker exec -e PGPASSWORD="$DB_PASSWORD" -i "$CONTAINER_NAME" \
    psql -X -q -U "$DB_USER" -d "$DB_NAME" "$@"
}

cleanup_host_processes() {
  for name in blocker blocked; do
    pid_file="$STATE_DIR/${name}.hostpid"
    if [[ -f "$pid_file" ]]; then
      host_pid="$(cat "$pid_file")"
      if kill -0 "$host_pid" >/dev/null 2>&1; then
        kill "$host_pid" >/dev/null 2>&1 || true
        wait "$host_pid" 2>/dev/null || true
      fi
      rm -f "$pid_file"
    fi
  done
}

cleanup_backend_processes() {
  for name in blocker blocked; do
    pid_file="$STATE_DIR/${name}.backendpid"
    if [[ -f "$pid_file" ]]; then
      backend_pid="$(cat "$pid_file")"
      run_psql -c "SELECT pg_terminate_backend(${backend_pid});" >/dev/null 2>&1 || true
      rm -f "$pid_file"
    fi
  done
}

start_demo() {
  cleanup_host_processes
  cleanup_backend_processes
  rm -f "$STATE_DIR"/*.log

  echo "Starting blocker session..."
  (
    run_psql <<SQL
\pset tuples_only on
\pset format unaligned
SELECT pg_backend_pid();
SELECT pg_advisory_lock(${LOCK_KEY});
SELECT pg_sleep(600);
SQL
  ) >"$STATE_DIR/blocker.log" 2>&1 &
  blocker_host_pid=$!
  echo "$blocker_host_pid" >"$STATE_DIR/blocker.hostpid"

  blocker_backend_pid=""
  for _ in $(seq 1 50); do
    if [[ -s "$STATE_DIR/blocker.log" ]]; then
      blocker_backend_pid="$(awk 'NF {print; exit}' "$STATE_DIR/blocker.log" | tr -d '[:space:]')"
      if [[ "$blocker_backend_pid" =~ ^[0-9]+$ ]]; then
        break
      fi
    fi
    sleep 0.1
  done

  if [[ -z "$blocker_backend_pid" ]]; then
    echo "Failed to capture blocker backend PID."
    exit 1
  fi

  echo "$blocker_backend_pid" >"$STATE_DIR/blocker.backendpid"
  echo "Blocker backend PID: $blocker_backend_pid"

  echo "Starting blocked session..."
  (
    run_psql <<SQL
\pset tuples_only on
\pset format unaligned
SELECT pg_backend_pid();
SELECT pg_advisory_lock(${LOCK_KEY});
SQL
  ) >"$STATE_DIR/blocked.log" 2>&1 &
  blocked_host_pid=$!
  echo "$blocked_host_pid" >"$STATE_DIR/blocked.hostpid"

  blocked_backend_pid=""
  for _ in $(seq 1 50); do
    if [[ -s "$STATE_DIR/blocked.log" ]]; then
      blocked_backend_pid="$(awk 'NF {print; exit}' "$STATE_DIR/blocked.log" | tr -d '[:space:]')"
      if [[ "$blocked_backend_pid" =~ ^[0-9]+$ ]]; then
        break
      fi
    fi
    sleep 0.1
  done

  if [[ -z "$blocked_backend_pid" ]]; then
    echo "Failed to capture blocked backend PID."
    exit 1
  fi

  echo "$blocked_backend_pid" >"$STATE_DIR/blocked.backendpid"
  echo "Blocked backend PID: $blocked_backend_pid"

  echo
  echo "The UI should now show a blocked chain:"
  echo "  blocked PID  = $blocked_backend_pid"
  echo "  blocking PID = $blocker_backend_pid"
  echo
  echo "Open /admin/database-sessions and click 'Terminate blocker'."
  echo "Run '$0 status' to verify current session state."
  echo "Run '$0 cleanup' if you want to remove the demo manually."
}

status_demo() {
  echo "Postgres sessions waiting on blockers in ${DB_NAME}:"
  run_psql <<'SQL'
\x off
SELECT
  blocked.pid AS blocked_pid,
  blocker.pid AS blocking_pid,
  blocked.wait_event_type,
  blocked.wait_event,
  left(regexp_replace(blocked.query, '\s+', ' ', 'g'), 80) AS blocked_query,
  left(regexp_replace(blocker.query, '\s+', ' ', 'g'), 80) AS blocking_query
FROM pg_stat_activity blocked
CROSS JOIN LATERAL unnest(pg_blocking_pids(blocked.pid)) AS blocker_pid
JOIN pg_stat_activity blocker ON blocker.pid = blocker_pid
WHERE blocked.datname = current_database()
ORDER BY blocked.pid, blocker.pid;
SQL
}

cleanup_demo() {
  cleanup_backend_processes
  cleanup_host_processes
  echo "Cleaned up demo sessions."
}

case "${1:-}" in
  start)
    start_demo
    ;;
  status)
    status_demo
    ;;
  cleanup)
    cleanup_demo
    ;;
  *)
    echo "Usage: $0 {start|status|cleanup}"
    exit 1
    ;;
esac
