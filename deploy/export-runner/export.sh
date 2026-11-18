#!/usr/bin/env bash
# Makes one data export of a Managed Trenova instance from its backups
# (decision 0026 in the control plane's documents), and nothing else:
#
#   1. finds the newest base backup in the instance's backups bucket;
#   2. restores it into this job's own volume and replays the archived WAL
#      to the end, with a PostgreSQL that listens on no network;
#   3. dumps every database (pg_dump, custom format) and the roles without
#      their passwords, and checks each dump lists (pg_restore --list);
#   4. copies the instance's documents (but the log bundles kept for Trenova
#      support);
#   5. stores one tar archive in the export's bucket, checks the bucket holds
#      every byte, and reports its size and SHA-256 in outcome.json.
#
# Any failure stops it at once and reports the step that failed in
# outcome.json; the control plane reads that, removes this job and tries
# again later. It never prints a credential: they are read from the
# environment by the tools, never passed on a command line or echoed.
#
# Configuration, all required (provider.RunnerEnv in the control plane):
# EXPORT_ID, INSTANCE_NAME, S3_ENDPOINT, S3_REGION, AWS_ACCESS_KEY_ID,
# AWS_SECRET_ACCESS_KEY, DOCUMENTS_BUCKET, BACKUPS_BUCKET, BACKUPS_PREFIX,
# SKIP_DOCUMENTS_PREFIX, EXPORT_BUCKET, ARCHIVE_KEY, OUTCOME_KEY.
set -euo pipefail
umask 077

readonly volume=/scratch
readonly runner=/opt/runner
# shellcheck source=lib.sh
. "${runner}/lib.sh"

# Started as root: the volume is handed to postgres, which PostgreSQL must
# run as, and the rest runs as postgres.
if [ "${1:-}" != "run" ]; then
  install -d -o postgres -g postgres -m 0700 "${volume}/work"
  exec su-exec postgres "$0" run
fi

readonly work="${volume}/work"
readonly pgdata="${work}/pgdata"
readonly socket="${work}/socket"
readonly out="${work}/export"
readonly log="${work}/postgres.log"
step="reading its configuration"
reason=""

say() { printf '%s %s\n' "$(date -u +%FT%TZ)" "$*" >&2; }

# fail stops the export with a reason staff can read.
fail() {
  reason="$1"
  exit 1
}

outcome() { python3 "${runner}/s3.py" put-json "${EXPORT_BUCKET}" "${OUTCOME_KEY}" "$1"; }

finish() {
  local code=$?
  if [ -f "${pgdata}/postmaster.pid" ]; then
    pg_ctl -D "${pgdata}" -m immediate -w stop >/dev/null 2>&1 || true
  fi
  if [ "${code}" -eq 0 ]; then
    say "the export is stored"
    return
  fi
  if [ -z "${reason}" ]; then
    reason="it stopped with exit code ${code}; the runner's log says more"
  fi
  say "failed while ${step}: ${reason}"
  # PostgreSQL's own words, for staff reading the runner's log: its log
  # names files and settings, never a credential.
  if [ -s "${log}" ]; then
    say "the last lines of PostgreSQL's log:"
    tail -n 30 "${log}" >&2 || true
  fi
  if ! outcome "$(jq -cn --arg step "${step}" --arg message "${reason}" '{ok: false, step: $step, message: $message}')"; then
    say "the failure could not be reported in the export's bucket"
  fi
  exit "${code}"
}
trap finish EXIT

for name in EXPORT_ID INSTANCE_NAME S3_ENDPOINT S3_REGION AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY DOCUMENTS_BUCKET \
  BACKUPS_BUCKET BACKUPS_PREFIX SKIP_DOCUMENTS_PREFIX EXPORT_BUCKET ARCHIVE_KEY OUTCOME_KEY; do
  if [ -z "${!name:-}" ]; then
    fail "${name} is not set"
  fi
done
say "export ${EXPORT_ID} started"

# Barman Cloud and boto3 read the credentials from the environment and the
# addressing style from this file: the buckets are addressed by path, as
# every provider's storage takes them.
export AWS_CONFIG_FILE="${work}/aws-config"
export AWS_DEFAULT_REGION="${S3_REGION}"
printf '[default]\nregion = %s\ns3 =\n    addressing_style = path\n' "${S3_REGION}" >"${AWS_CONFIG_FILE}"
readonly source_url="s3://${BACKUPS_BUCKET}/${BACKUPS_PREFIX%/}"
rm -rf "${pgdata}" "${out}" "${socket}"
mkdir -p "${socket}" "${out}/database"

step="finding the newest backup"
server=""
backup=""
backup_ended=""
servers="$(python3 "${runner}/s3.py" servers "${BACKUPS_BUCKET}" "${BACKUPS_PREFIX}")"
for candidate in ${servers}; do
  listed="$(barman-cloud-backup-list --format json --endpoint-url "${S3_ENDPOINT}" "${source_url}" "${candidate}")"
  # Barman's backup IDs are their start times (20261116T030000), so the
  # greatest is the newest.
  newest="$(jq -r '[.backups_list[] | select(.status == "DONE")] | sort_by(.backup_id) | last // empty
    | "\(.backup_id) \(.end_time // "")"' <<<"${listed}")"
  if [ -n "${newest}" ] && [[ "${newest%% *}" > "${backup}" ]]; then
    server="${candidate}"
    backup="${newest%% *}"
    backup_ended="${newest#* }"
  fi
done
if [ -z "${backup}" ]; then
  fail "the backups bucket holds no completed base backup"
fi
say "restoring backup ${backup} of ${server} (ended ${backup_ended})"

step="restoring the database from its backup"
barman-cloud-restore --endpoint-url "${S3_ENDPOINT}" "${source_url}" "${server}" "${backup}" "${pgdata}"
chmod 0700 "${pgdata}"

step="preparing the restored database"
# The backup carries CloudNativePG's configuration, which names its
# certificates and its own tools; this server gets its own: no network, no
# archiving, and at least the primary's limits, which recovery requires.
controldata="$(pg_controldata "${pgdata}")"
setting() {
  local value
  value="$(sed -n "s/^$1 setting: *//p" <<<"${controldata}")"
  if ! [[ "${value}" =~ ^[0-9]+$ ]]; then
    fail "the backup's control file has no $1 setting"
  fi
  printf -v "$2" '%s' "${value}"
}
setting max_connections max_connections
setting max_worker_processes max_worker_processes
setting max_wal_senders max_wal_senders
setting max_prepared_xacts max_prepared_xacts
setting max_locks_per_xact max_locks_per_xact
rm -f "${pgdata}/standby.signal" "${pgdata}/postmaster.pid" "${pgdata}/postmaster.opts" \
  "${pgdata}/custom.conf" "${pgdata}/override.conf"
: >"${pgdata}/postgresql.auto.conf"
cat >"${pgdata}/postgresql.conf" <<CONF
listen_addresses = ''
unix_socket_directories = '${socket}'
max_connections = ${max_connections}
max_worker_processes = ${max_worker_processes}
max_wal_senders = ${max_wal_senders}
max_prepared_transactions = ${max_prepared_xacts}
max_locks_per_transaction = ${max_locks_per_xact}
wal_level = replica
hot_standby = off
archive_mode = off
shared_buffers = 512MB
maintenance_work_mem = 256MB
logging_collector = off
restore_command = 'barman-cloud-wal-restore --endpoint-url ${S3_ENDPOINT} ${source_url} ${server} %f %p'
recovery_target_timeline = 'latest'
recovery_target_action = 'promote'
CONF
printf 'local all all trust\n' >"${pgdata}/pg_hba.conf"
: >"${pgdata}/pg_ident.conf"
touch "${pgdata}/recovery.signal"

step="replaying the database's changes since the backup"
# With hot_standby off the server accepts connections only once it has
# replayed every archived change and promoted itself. pg_ctl -w returns as
# soon as recovery begins (a standby without hot standby counts as started),
# so wait here for the server to take connections: at most 12 hours.
pg_ctl -D "${pgdata}" -l "${log}" -w -t 600 start
waited=0
wait_for_promotion "${pgdata}" "${socket}" 43200 || waited=$?
case "${waited}" in
  0) ;;
  1) fail "the restored database stopped before it finished replaying its changes" ;;
  *) fail "the restored database did not finish replaying its changes in 12 hours" ;;
esac
data_as_of=""
replayed="$(sed -n 's/.*last completed transaction was at log time \(.*\)$/\1/p' "${log}" | tail -n 1)"
if [ -n "${replayed}" ]; then
  data_as_of="$(date -u -d "${replayed}" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || true)"
fi
server_version="$(psql -h "${socket}" -U postgres -d postgres -XAtc "SHOW server_version")"

step="dumping the database"
databases=()
while IFS= read -r db; do
  if [ "${db}" = "postgres" ]; then
    tables="$(psql -h "${socket}" -U postgres -d postgres -XAtc "SELECT count(*) FROM pg_class c
      JOIN pg_namespace n ON n.oid = c.relnamespace
      WHERE c.relkind IN ('r', 'p') AND n.nspname NOT IN ('pg_catalog', 'information_schema')")"
    if [ "${tables}" = "0" ]; then
      continue
    fi
  fi
  if ! [[ "${db}" =~ ^[A-Za-z0-9_][A-Za-z0-9_-]{0,62}$ ]]; then
    fail "a database's name cannot be used as a file name"
  fi
  databases+=("${db}")
done < <(psql -h "${socket}" -U postgres -d postgres -XAtc \
  "SELECT datname FROM pg_database WHERE datallowconn AND NOT datistemplate ORDER BY datname")
if [ "${#databases[@]}" -eq 0 ]; then
  fail "the restored server holds no database to export"
fi
extensions='{}'
for db in "${databases[@]}"; do
  # The extensions a restore needs installed, for the README and manifest.
  installed="$(psql -h "${socket}" -U postgres -d "${db}" -XAtc \
    "SELECT coalesce(json_agg(extname ORDER BY extname), '[]') FROM pg_extension WHERE extname <> 'plpgsql'")"
  extensions="$(jq -c --arg db "${db}" --argjson installed "${installed}" '. + {($db): $installed}' <<<"${extensions}")"
  step="dumping the database ${db}"
  pg_dump -h "${socket}" -U postgres -Fc -f "${out}/database/${db}.dump" "${db}"
  step="checking the dump of ${db}"
  pg_restore --list "${out}/database/${db}.dump" >/dev/null
done
step="dumping the database roles"
pg_dumpall -h "${socket}" -U postgres --globals-only --no-role-passwords -f "${out}/database/roles.sql"
pg_ctl -D "${pgdata}" -m fast -w stop
rm -rf "${pgdata}"

step="copying the documents"
documents="$(python3 "${runner}/s3.py" download "${DOCUMENTS_BUCKET}" "${out}/documents" "${SKIP_DOCUMENTS_PREFIX}")"
say "copied ${documents} documents"

step="writing the README and the manifest"
created="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
{
  printf 'Trenova data export of %s, made %s (UTC).\n\n' "${INSTANCE_NAME}" "$(date -u '+%-d %B %Y at %H:%M')"
  cat "${runner}/README.txt"
} >"${out}/README.txt"
jq -n --arg export "${EXPORT_ID}" --arg instance "${INSTANCE_NAME}" --arg created "${created}" \
  --arg asOf "${data_as_of}" --arg backup "${backup}" --arg server "${server}" --arg version "${server_version}" \
  --argjson documents "${documents}" --argjson extensions "${extensions}" --args '{
    exportId: $export, instance: $instance, createdAt: $created,
    dataAsOf: (if $asOf == "" then null else $asOf end),
    postgresVersion: $version, backup: {server: $server, id: $backup},
    databases: [$ARGS.positional[] | {name: ., file: ("database/" + . + ".dump"), extensions: $extensions[.]}],
    roles: "database/roles.sql", documents: {folder: "documents", count: $documents}
  }' "${databases[@]}" >"${out}/manifest.json"

step="storing the archive"
stored="$(tar -C "${out}" -cf - README.txt manifest.json database documents |
  python3 "${runner}/s3.py" put-stream "${EXPORT_BUCKET}" "${ARCHIVE_KEY}")"
size="$(jq -r '.size' <<<"${stored}")"
sha256="$(jq -r '.sha256' <<<"${stored}")"

step="checking the stored archive"
in_bucket="$(python3 "${runner}/s3.py" size "${EXPORT_BUCKET}" "${ARCHIVE_KEY}")"
if [ "${in_bucket}" != "${size}" ]; then
  fail "the bucket holds ${in_bucket} bytes of an archive of ${size}"
fi
rm -rf "${out}"

step="reporting the outcome"
outcome "$(jq -cn --arg key "${ARCHIVE_KEY}" --argjson size "${size}" --arg sha "${sha256}" --arg asOf "${data_as_of}" \
  --argjson documents "${documents}" --args \
  '{ok: true, archiveKey: $key, sizeBytes: $size, sha256: $sha, dataAsOf: $asOf, documents: $documents,
    databases: $ARGS.positional}' "${databases[@]}")"
