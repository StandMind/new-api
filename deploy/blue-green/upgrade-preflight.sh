#!/usr/bin/env bash
set -Eeuo pipefail

DEPLOY_PATH="${DEPLOY_PATH:-/opt/new-api-stack}"
BASE_ENV_FILE="${BASE_ENV_FILE:-${DEPLOY_PATH}/.env}"
REPORT_FILE="${PREFLIGHT_REPORT_FILE:?PREFLIGHT_REPORT_FILE is required}"
POSTGRES_IMAGE="${PREFLIGHT_POSTGRES_IMAGE:-postgres:15-alpine}"
REDIS_IMAGE="${PREFLIGHT_REDIS_IMAGE:-redis:7-alpine}"
PREFIX="${PREFLIGHT_PREFIX:-new-api-upgrade-preflight}"
NETWORK="${PREFIX}-net"
POSTGRES_CONTAINER="${PREFIX}-postgres"
REDIS_CONTAINER="${PREFIX}-redis"
OLD_SLAVE_CONTAINER="${PREFIX}-old-slave"

CANDIDATE_IMAGE="${1:-}"
OLD_MASTER_IMAGE="${2:-}"
OLD_SLAVE_IMAGE="${3:-}"
BACKUP_FILE="${4:-}"

REPORT_DIR="$(dirname "${REPORT_FILE}")"
SCHEMA_BEFORE="${REPORT_FILE%.txt}.schema-before.sql"
SCHEMA_AFTER="${REPORT_FILE%.txt}.schema-after.sql"
SCHEMA_FINAL="${REPORT_FILE%.txt}.schema-final.sql"
SCHEMA_DIFF="${REPORT_FILE%.txt}.schema.diff"
FINAL_DIFF="${REPORT_FILE%.txt}.old-master.diff"
CATALOG_BEFORE="${REPORT_FILE%.txt}.catalog-before.tsv"
CATALOG_AFTER="${REPORT_FILE%.txt}.catalog-after.tsv"
CATALOG_FINAL="${REPORT_FILE%.txt}.catalog-final.tsv"
CATALOG_REMOVED="${REPORT_FILE%.txt}.catalog-removed.tsv"
FINAL_CATALOG_DIFF="${REPORT_FILE%.txt}.old-master-catalog.diff"

log() {
  printf '[upgrade-preflight] %s\n' "$*" | tee -a "${REPORT_FILE}"
}

fatal() {
  log "ERROR: $*"
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fatal "$1 is required"
}

env_value() {
  local key="$1"
  local fallback="$2"
  local value
  value="$(awk -F= -v key="${key}" '
    $1 == key {
      sub(/^[^=]*=/, "")
      gsub(/\r$/, "")
      print
      exit
    }
  ' "${BASE_ENV_FILE}")"
  value="${value%\"}"
  value="${value#\"}"
  value="${value%\'}"
  value="${value#\'}"
  if [ -z "${value}" ]; then
    value="${fallback}"
  fi
  printf '%s\n' "${value}"
}

cleanup() {
  local container
  for container in \
    "${PREFIX}-candidate-master" \
    "${PREFIX}-candidate-slave" \
    "${PREFIX}-old-master" \
    "${OLD_SLAVE_CONTAINER}" \
    "${REDIS_CONTAINER}" \
    "${POSTGRES_CONTAINER}"; do
    docker rm -f "${container}" >/dev/null 2>&1 || true
  done
  docker network rm "${NETWORK}" >/dev/null 2>&1 || true
}

# The image entrypoint briefly exposes an init-time server before PID 1 becomes postgres.
postgres_is_ready() {
  local container="$1"

  docker exec "${container}" sh -c \
    '[ "$(cat /proc/1/comm)" = postgres ]' >/dev/null 2>&1 \
    && docker exec "${container}" psql \
      -X -v ON_ERROR_STOP=1 -At \
      -U "${POSTGRES_USER}" \
      -d "${POSTGRES_DB}" \
      -c 'SELECT 1' >/dev/null 2>&1
}

wait_postgres() {
  for _ in $(seq 1 60); do
    if postgres_is_ready "${POSTGRES_CONTAINER}"; then
      return
    fi
    if [ "$(docker inspect -f '{{.State.Running}}' "${POSTGRES_CONTAINER}" 2>/dev/null || true)" != "true" ]; then
      docker logs "${POSTGRES_CONTAINER}" >> "${REPORT_FILE}" 2>&1 || true
      fatal "isolated PostgreSQL exited before becoming ready"
    fi
    sleep 1
  done
  docker logs "${POSTGRES_CONTAINER}" >> "${REPORT_FILE}" 2>&1 || true
  fatal "isolated PostgreSQL did not become ready"
}

wait_app() {
  local container="$1"
  local path="$2"
  for _ in $(seq 1 90); do
    if docker exec "${container}" wget -q -O - \
      "http://127.0.0.1:3000${path}" 2>/dev/null \
      | grep -q '"success"[[:space:]]*:[[:space:]]*true'; then
      return
    fi
    if [ "$(docker inspect -f '{{.State.Running}}' "${container}" 2>/dev/null || true)" != "true" ]; then
      docker logs "${container}" >> "${REPORT_FILE}" 2>&1 || true
      fatal "${container} exited before ${path} became healthy"
    fi
    sleep 2
  done
  docker logs "${container}" >> "${REPORT_FILE}" 2>&1 || true
  fatal "timed out waiting for ${container}${path}"
}

start_app() {
  local container="$1"
  local image="$2"
  local node_type="$3"
  local health_path="$4"

  docker run -d \
    --name "${container}" \
    --network "${NETWORK}" \
    --env-file "${BASE_ENV_FILE}" \
    -e "SQL_DSN=${PREFLIGHT_SQL_DSN}" \
    -e LOG_SQL_DSN= \
    -e "REDIS_CONN_STRING=redis://${REDIS_CONTAINER}:6379" \
    -e "NODE_TYPE=${node_type}" \
    -e "NODE_NAME=${container}" \
    -e PORT=3000 \
    -e BATCH_UPDATE_ENABLED=false \
    -e PRE_SHUTDOWN_DRAIN_SECONDS=0 \
    -e SHUTDOWN_TIMEOUT_SECONDS=30 \
    --tmpfs /data \
    --tmpfs /app/logs \
    "${image}" --log-dir /app/logs >/dev/null

  wait_app "${container}" "${health_path}"
  log "compatibility check passed: ${container} image=${image} path=${health_path}"
}

stop_app() {
  local container="$1"
  docker stop --time 35 "${container}" >/dev/null 2>&1 || true
  docker rm "${container}" >/dev/null 2>&1 || true
}

run_candidate_migration() {
  docker run --rm \
    --network "${NETWORK}" \
    --env-file "${BASE_ENV_FILE}" \
    -e "SQL_DSN=${PREFLIGHT_SQL_DSN}" \
    -e LOG_SQL_DSN= \
    -e "REDIS_CONN_STRING=redis://${REDIS_CONTAINER}:6379" \
    -e NODE_TYPE=master \
    -e NODE_NAME=upgrade-preflight-migration \
    -e BATCH_UPDATE_ENABLED=false \
    --tmpfs /data \
    --tmpfs /app/logs \
    "${CANDIDATE_IMAGE}" --migrate-only --log-dir /app/logs \
    >> "${REPORT_FILE}" 2>&1
}

dump_schema() {
  local output="$1"
  docker exec "${POSTGRES_CONTAINER}" pg_dump \
    --schema-only \
    --no-owner \
    --no-privileges \
    -U "${POSTGRES_USER}" \
    -d "${POSTGRES_DB}" \
    | sed -e '/^\\restrict /d' -e '/^\\unrestrict /d' > "${output}"
}

dump_catalog() {
  local output="$1"
  docker exec -i "${POSTGRES_CONTAINER}" psql \
    -X -v ON_ERROR_STOP=1 -At -F $'\t' \
    -U "${POSTGRES_USER}" \
    -d "${POSTGRES_DB}" <<'SQL' \
    | LC_ALL=C sort -u > "${output}"
SELECT 'table', n.nspname, c.relname, '',
       encode(convert_to(c.relkind::text, 'UTF8'), 'hex')
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind IN ('r', 'p', 'S', 'v', 'm', 'f')
  AND n.nspname <> 'information_schema'
  AND n.nspname !~ '^pg_'
UNION ALL
SELECT 'column', table_schema, table_name, column_name,
       encode(convert_to(concat_ws(E'\x1f',
         ordinal_position::text,
         CASE WHEN approved_type_change THEN '<approved-type>' ELSE data_type END,
         CASE WHEN approved_type_change THEN '<approved-type>' ELSE udt_schema END,
         CASE WHEN approved_type_change THEN '<approved-type>' ELSE udt_name END,
         CASE WHEN approved_type_change THEN '<approved-type>' ELSE coalesce(character_maximum_length::text, '') END,
         CASE WHEN approved_type_change THEN '<approved-type>' ELSE coalesce(numeric_precision::text, '') END,
         CASE WHEN approved_type_change THEN '<approved-type>' ELSE coalesce(numeric_scale::text, '') END,
         is_nullable,
         coalesce(column_default, '')
       ), 'UTF8'), 'hex')
FROM (
  SELECT c.*,
         (table_name = 'subscription_plans' AND column_name = 'price_amount')
         OR (table_name = 'tokens' AND column_name = 'model_limits') AS approved_type_change
  FROM information_schema.columns AS c
) AS column_inventory
WHERE table_schema <> 'information_schema'
  AND table_schema !~ '^pg_'
UNION ALL
SELECT 'constraint', n.nspname, c.relname, con.conname,
       encode(convert_to(concat_ws(E'\x1f',
         con.contype::text,
         con.condeferrable::text,
         con.condeferred::text,
         pg_catalog.pg_get_constraintdef(con.oid, true)
       ), 'UTF8'), 'hex')
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_class c ON c.oid = con.conrelid
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname <> 'information_schema'
  AND n.nspname !~ '^pg_'
UNION ALL
SELECT 'index', schemaname, tablename, indexname,
       encode(convert_to(indexdef, 'UTF8'), 'hex')
FROM pg_catalog.pg_indexes
WHERE schemaname <> 'information_schema'
  AND schemaname !~ '^pg_'
UNION ALL
SELECT 'trigger', n.nspname, c.relname, t.tgname,
       encode(convert_to(pg_catalog.pg_get_triggerdef(t.oid, true), 'UTF8'), 'hex')
FROM pg_catalog.pg_trigger t
JOIN pg_catalog.pg_class c ON c.oid = t.tgrelid
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE NOT t.tgisinternal
  AND n.nspname <> 'information_schema'
  AND n.nspname !~ '^pg_'
UNION ALL
SELECT 'view', n.nspname, c.relname, '',
       encode(convert_to(pg_catalog.pg_get_viewdef(c.oid, true), 'UTF8'), 'hex')
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind IN ('v', 'm')
  AND n.nspname <> 'information_schema'
  AND n.nspname !~ '^pg_'
UNION ALL
SELECT 'sequence', n.nspname, c.relname, '',
       encode(convert_to(concat_ws(E'\x1f',
         s.seqstart::text,
         s.seqincrement::text,
         s.seqmax::text,
         s.seqmin::text,
         s.seqcache::text,
         s.seqcycle::text
       ), 'UTF8'), 'hex')
FROM pg_catalog.pg_sequence s
JOIN pg_catalog.pg_class c ON c.oid = s.seqrelid
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname <> 'information_schema'
  AND n.nspname !~ '^pg_';
SQL
}

column_data_type() {
  local table_name="$1"
  local column_name="$2"
  docker exec "${POSTGRES_CONTAINER}" psql \
    -X -At -v ON_ERROR_STOP=1 \
    -U "${POSTGRES_USER}" \
    -d "${POSTGRES_DB}" \
    -c "SELECT data_type FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = '${table_name}' AND column_name = '${column_name}'"
}

validate_expand_only_diff() {
  local schema_before="$1"
  local schema_after="$2"
  local schema_diff="$3"
  local catalog_before="$4"
  local catalog_after="$5"
  local catalog_removed="$6"
  local line
  local rejected=0

  if diff -u "${schema_before}" "${schema_after}" > "${schema_diff}"; then
    log "schema diff is empty"
  else
    log "raw schema diff saved to ${schema_diff}"
  fi

  comm -23 "${catalog_before}" "${catalog_after}" > "${catalog_removed}"
  while IFS= read -r line; do
    [ -n "${line}" ] || continue
    printf 'removed_or_changed_catalog_object=%s\n' "${line}" >> "${REPORT_FILE}"
    rejected=1
  done < "${catalog_removed}"

  [ "${rejected}" -eq 0 ] \
    || fatal "catalog diff contains a removed object or a non-type change to an existing object"
  [ "$(column_data_type subscription_plans price_amount)" = "numeric" ] \
    || fatal "subscription_plans.price_amount is not numeric after migration"
  [ "$(column_data_type tokens model_limits)" = "text" ] \
    || fatal "tokens.model_limits is not text after migration"
  log "catalog diff is additive except for approved price_amount/model_limits type changes"
}

main() {
  [ -n "${CANDIDATE_IMAGE}" ] || fatal "candidate image is required"
  [ -n "${OLD_MASTER_IMAGE}" ] || fatal "old master image is required"
  [ -n "${OLD_SLAVE_IMAGE}" ] || fatal "old slave image is required"
  [ -n "${BACKUP_FILE}" ] || fatal "backup file is required"

  require_command awk
  require_command comm
  require_command diff
  require_command docker
  require_command gzip
  require_command grep
  require_command sed
  require_command tee

  [ -f "${BASE_ENV_FILE}" ] || fatal "${BASE_ENV_FILE} is missing"
  [ -f "${BACKUP_FILE}" ] || fatal "${BACKUP_FILE} is missing"

  install -d -m 700 "${REPORT_DIR}"
  : > "${REPORT_FILE}"
  chmod 600 "${REPORT_FILE}"

  POSTGRES_USER="$(env_value POSTGRES_USER postgres)"
  POSTGRES_DB="$(env_value POSTGRES_DB new-api)"
  [[ "${POSTGRES_USER}" =~ ^[A-Za-z0-9_][A-Za-z0-9_-]*$ ]] \
    || fatal "POSTGRES_USER contains unsupported characters"
  [[ "${POSTGRES_DB}" =~ ^[A-Za-z0-9_][A-Za-z0-9_-]*$ ]] \
    || fatal "POSTGRES_DB contains unsupported characters"
  PREFLIGHT_SQL_DSN="postgresql://${POSTGRES_USER}@${POSTGRES_CONTAINER}:5432/${POSTGRES_DB}?sslmode=disable"

  trap cleanup EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
  cleanup
  gzip -t "${BACKUP_FILE}" || fatal "backup gzip integrity check failed"

  log "started=$(date -Ins)"
  log "candidate_image=${CANDIDATE_IMAGE}"
  log "old_master_image=${OLD_MASTER_IMAGE}"
  log "old_slave_image=${OLD_SLAVE_IMAGE}"
  log "backup_file=${BACKUP_FILE}"

  docker network create --internal "${NETWORK}" >/dev/null
  docker run -d \
    --name "${POSTGRES_CONTAINER}" \
    --network "${NETWORK}" \
    -e POSTGRES_HOST_AUTH_METHOD=trust \
    -e "POSTGRES_USER=${POSTGRES_USER}" \
    -e "POSTGRES_DB=${POSTGRES_DB}" \
    "${POSTGRES_IMAGE}" >/dev/null
  docker run -d \
    --name "${REDIS_CONTAINER}" \
    --network "${NETWORK}" \
    "${REDIS_IMAGE}" >/dev/null
  wait_postgres

  gzip -dc "${BACKUP_FILE}" \
    | docker exec -i "${POSTGRES_CONTAINER}" psql \
      -v ON_ERROR_STOP=1 \
      -U "${POSTGRES_USER}" \
      -d "${POSTGRES_DB}" >> "${REPORT_FILE}" 2>&1
  log "production backup restored into isolated PostgreSQL 15"

  docker exec "${POSTGRES_CONTAINER}" psql \
    -v ON_ERROR_STOP=1 \
    -U "${POSTGRES_USER}" \
    -d "${POSTGRES_DB}" \
    -c 'CREATE TABLE IF NOT EXISTS upgrade_preflight_writes (id bigserial PRIMARY KEY, created_at timestamptz NOT NULL DEFAULT now())' \
    >> "${REPORT_FILE}" 2>&1
  dump_schema "${SCHEMA_BEFORE}"
  dump_catalog "${CATALOG_BEFORE}"

  start_app "${OLD_SLAVE_CONTAINER}" "${OLD_SLAVE_IMAGE}" slave /api/status
  docker exec -d "${POSTGRES_CONTAINER}" sh -c \
    'while :; do psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "INSERT INTO upgrade_preflight_writes DEFAULT VALUES" >/dev/null 2>&1; sleep 0.2; done'
  docker exec "${OLD_SLAVE_CONTAINER}" sh -c \
    'rm -f /tmp/upgrade-preflight-read-count /tmp/upgrade-preflight-read-failed'
  docker exec -d "${OLD_SLAVE_CONTAINER}" sh -c \
    'while :; do wget -q -O /dev/null http://127.0.0.1:3000/api/status || { touch /tmp/upgrade-preflight-read-failed; exit 1; }; printf "." >> /tmp/upgrade-preflight-read-count; sleep 0.2; done'

  sleep 1
  local read_count_before
  local read_count_after
  local write_count_before
  local write_count_after
  read_count_before="$(docker exec "${OLD_SLAVE_CONTAINER}" sh -c 'wc -c < /tmp/upgrade-preflight-read-count')"
  write_count_before="$(docker exec "${POSTGRES_CONTAINER}" psql \
    -At -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" \
    -c 'SELECT count(*) FROM upgrade_preflight_writes')"

  run_candidate_migration
  log "candidate migrate-only pass 1 completed while old-version reads and writes continued"
  run_candidate_migration
  log "candidate migrate-only pass 2 completed (idempotency)"

  docker exec "${OLD_SLAVE_CONTAINER}" test ! -e /tmp/upgrade-preflight-read-failed \
    || fatal "old-version API reads failed during migration"
  read_count_after="$(docker exec "${OLD_SLAVE_CONTAINER}" sh -c 'wc -c < /tmp/upgrade-preflight-read-count')"
  write_count_after="$(docker exec "${POSTGRES_CONTAINER}" psql \
    -At -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" \
    -c 'SELECT count(*) FROM upgrade_preflight_writes')"
  [ "${read_count_after}" -gt "${read_count_before}" ] \
    || fatal "old-version API reads did not progress during migration"
  [ "${write_count_after}" -gt "${write_count_before}" ] \
    || fatal "simulated writes did not progress during migration"
  log "old_api_read_count_before=${read_count_before} old_api_read_count_after=${read_count_after}"
  log "simulated_write_count_before=${write_count_before} simulated_write_count_after=${write_count_after}"

  dump_schema "${SCHEMA_AFTER}"
  dump_catalog "${CATALOG_AFTER}"
  validate_expand_only_diff \
    "${SCHEMA_BEFORE}" \
    "${SCHEMA_AFTER}" \
    "${SCHEMA_DIFF}" \
    "${CATALOG_BEFORE}" \
    "${CATALOG_AFTER}" \
    "${CATALOG_REMOVED}"
  stop_app "${OLD_SLAVE_CONTAINER}"

  start_app "${PREFIX}-candidate-master" "${CANDIDATE_IMAGE}" master /readyz
  stop_app "${PREFIX}-candidate-master"
  start_app "${PREFIX}-candidate-slave" "${CANDIDATE_IMAGE}" slave /readyz
  stop_app "${PREFIX}-candidate-slave"
  start_app "${OLD_SLAVE_CONTAINER}" "${OLD_SLAVE_IMAGE}" slave /api/status
  stop_app "${OLD_SLAVE_CONTAINER}"
  start_app "${PREFIX}-old-master" "${OLD_MASTER_IMAGE}" master /api/status
  stop_app "${PREFIX}-old-master"

  dump_schema "${SCHEMA_FINAL}"
  dump_catalog "${CATALOG_FINAL}"
  if ! diff -u "${SCHEMA_AFTER}" "${SCHEMA_FINAL}" > "${FINAL_DIFF}"; then
    fatal "old master changed the candidate-migrated schema; inspect ${FINAL_DIFF}"
  fi
  if ! diff -u "${CATALOG_AFTER}" "${CATALOG_FINAL}" > "${FINAL_CATALOG_DIFF}"; then
    fatal "old master changed the candidate-migrated catalog; inspect ${FINAL_CATALOG_DIFF}"
  fi

  log "completed=$(date -Ins)"
  log "result=success"
}

if [ "${PREFLIGHT_LIB_ONLY:-false}" != "true" ]; then
  main "$@"
fi
