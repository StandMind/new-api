#!/usr/bin/env bash
set -Eeuo pipefail

DEPLOY_PATH="${DEPLOY_PATH:-/opt/new-api-stack}"
COMPOSE_FILE="${COMPOSE_FILE:-${DEPLOY_PATH}/docker-compose.slots.yml}"
BASE_ENV_FILE="${BASE_ENV_FILE:-${DEPLOY_PATH}/.env}"
SLOT_ENV_FILE="${SLOT_ENV_FILE:-${DEPLOY_PATH}/slots.env}"
ACTIVE_SLOT_FILE="${ACTIVE_SLOT_FILE:-${DEPLOY_PATH}/active-slot}"
CADDYFILE="${CADDYFILE:-/opt/aivrae-caddy/Caddyfile}"
CADDY_CONTAINER="${CADDY_CONTAINER:-aivrae-caddy}"
LOCK_FILE="${LOCK_FILE:-/var/lock/new-api-blue-green.lock}"
PUBLIC_STATUS_URL="${PUBLIC_STATUS_URL:-https://aivrae.com/api/status}"
PUBLIC_MODELS_URL="${PUBLIC_MODELS_URL:-https://aivrae.com/v1/models}"
PUBLIC_CHAT_URL="${PUBLIC_CHAT_URL:-https://aivrae.com/v1/chat/completions}"
SMOKE_TOKEN="${SMOKE_TOKEN:-}"
SMOKE_MODEL="${SMOKE_MODEL:-}"
DEPLOY_REGISTRY_USERNAME="${DEPLOY_REGISTRY_USERNAME:-}"
DEPLOY_REGISTRY_TOKEN="${DEPLOY_REGISTRY_TOKEN:-}"
DEPLOY_HISTORY_FILE="${DEPLOY_HISTORY_FILE:-${DEPLOY_PATH}/deployment-history.log}"
UPGRADE_STATE_FILE="${UPGRADE_STATE_FILE:-${DEPLOY_PATH}/upgrade-state}"
PREFLIGHT_REPORT_DIR="${PREFLIGHT_REPORT_DIR:-${DEPLOY_PATH}/preflight-reports}"
BACKUP_DIR="${BACKUP_DIR:-${DEPLOY_PATH}/backups}"
UPGRADE_OBSERVE_SECONDS="${UPGRADE_OBSERVE_SECONDS:-3600}"
UPGRADE_OBSERVE_INTERVAL="${UPGRADE_OBSERVE_INTERVAL:-5}"
UPGRADE_RETAIN_SECONDS="${UPGRADE_RETAIN_SECONDS:-86400}"
UPGRADE_ZERO_CONNECTION_SECONDS="${UPGRADE_ZERO_CONNECTION_SECONDS:-600}"
UPGRADE_EXTERNAL_PROBE_SECONDS="${UPGRADE_EXTERNAL_PROBE_SECONDS:-$((UPGRADE_OBSERVE_SECONDS + 7200))}"
CAPTURE_CADDY_BACKUP_TO_STATE="false"
DRAIN_SSE_PID=""
DRAIN_SSE_FILE=""
EXTERNAL_PROBE_PID=""
EXTERNAL_PROBE_FILE=""

usage() {
  cat <<'EOF'
Usage:
  deploy-blue-green.sh status
  deploy-blue-green.sh switch <blue|green>
  deploy-blue-green.sh deploy <image@sha256:digest>
  deploy-blue-green.sh preflight-upgrade <image@sha256:digest>
  deploy-blue-green.sh start-upgrade <image@sha256:digest>
  deploy-blue-green.sh finalize-upgrade
  deploy-blue-green.sh rollback-upgrade

The deploy command recreates only the inactive slot. It refuses to proceed if
that slot still has established HTTP connections. The previously active slot
is kept running after the Caddy switch.
EOF
}

log() {
  printf '[blue-green] %s\n' "$*"
}

fatal() {
  printf '[blue-green] ERROR: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fatal "$1 is required"
}

validate_image_ref() {
  local image_ref="$1"
  [[ "${image_ref}" =~ ^[A-Za-z0-9._/:@-]+@sha256:[a-f0-9]{64}$ ]] \
    || fatal "image must be an immutable image@sha256:digest reference"
}

validate_slot() {
  case "$1" in
    blue|green) ;;
    *) fatal "slot must be blue or green" ;;
  esac
}

service_for_slot() {
  printf 'new-api-%s\n' "$1"
}

other_slot() {
  if [ "$1" = "blue" ]; then
    printf 'green\n'
  else
    printf 'blue\n'
  fi
}

image_key_for_slot() {
  printf '%s_IMAGE\n' "$(printf '%s' "$1" | tr '[:lower:]' '[:upper:]')"
}

health_key_for_slot() {
  printf '%s_HEALTH_PATH\n' "$(printf '%s' "$1" | tr '[:lower:]' '[:upper:]')"
}

compose() {
  docker compose \
    --env-file "${BASE_ENV_FILE}" \
    --env-file "${SLOT_ENV_FILE}" \
    -f "${COMPOSE_FILE}" \
    "$@"
}

registry_login() {
  if [ -z "${DEPLOY_REGISTRY_TOKEN}" ]; then
    return
  fi
  [ -n "${DEPLOY_REGISTRY_USERNAME}" ] \
    || fatal "DEPLOY_REGISTRY_USERNAME is required when DEPLOY_REGISTRY_TOKEN is set"
  printf '%s' "${DEPLOY_REGISTRY_TOKEN}" \
    | docker login ghcr.io -u "${DEPLOY_REGISTRY_USERNAME}" --password-stdin >/dev/null
}

read_active_slot() {
  [ -s "${ACTIVE_SLOT_FILE}" ] || fatal "${ACTIVE_SLOT_FILE} is missing"
  local slot
  slot="$(tr -d '[:space:]' < "${ACTIVE_SLOT_FILE}")"
  validate_slot "${slot}"
  printf '%s\n' "${slot}"
}

write_active_slot() {
  local slot="$1"
  local candidate="${ACTIVE_SLOT_FILE}.candidate"
  validate_slot "${slot}"
  printf '%s\n' "${slot}" > "${candidate}"
  chmod 600 "${candidate}"
  mv "${candidate}" "${ACTIVE_SLOT_FILE}"
}

record_switch() {
  local active_slot="$1"
  local previous_slot="$2"
  local active_service
  local previous_service
  local active_image
  local previous_image
  active_service="$(service_for_slot "${active_slot}")"
  previous_service="$(service_for_slot "${previous_slot}")"
  active_image="$(docker inspect -f '{{.Config.Image}}' "${active_service}")"
  previous_image="$(docker inspect -f '{{.Config.Image}}' "${previous_service}")"

  printf '%s active_slot=%s active_image=%s previous_slot=%s previous_image=%s\n' \
    "$(date -Ins)" \
    "${active_slot}" \
    "${active_image}" \
    "${previous_slot}" \
    "${previous_image}" \
    >> "${DEPLOY_HISTORY_FILE}"
  chmod 600 "${DEPLOY_HISTORY_FILE}"
}

update_env_value() {
  local key="$1"
  local value="$2"
  local candidate="${SLOT_ENV_FILE}.candidate"

  awk -F= -v key="${key}" -v value="${value}" '
    BEGIN { updated = 0 }
    $1 == key { print key "=" value; updated = 1; next }
    { print }
    END { if (!updated) print key "=" value }
  ' "${SLOT_ENV_FILE}" > "${candidate}"
  chmod 600 "${candidate}"
  mv "${candidate}" "${SLOT_ENV_FILE}"
}

update_slot_image() {
  local slot="$1"
  local image_ref="$2"
  update_env_value "$(image_key_for_slot "${slot}")" "${image_ref}"
}

update_slot_health_path() {
  local slot="$1"
  local health_path="$2"
  update_env_value "$(health_key_for_slot "${slot}")" "${health_path}"
}

read_env_value() {
  local key="$1"
  awk -F= -v key="${key}" '$1 == key { sub(/^[^=]*=/, ""); print; exit }' \
    "${SLOT_ENV_FILE}"
}

slot_health_path() {
  local value
  value="$(read_env_value "$(health_key_for_slot "$1")")"
  printf '%s\n' "${value:-/api/status}"
}

state_get() {
  local key="$1"
  [ -f "${UPGRADE_STATE_FILE}" ] || return 0
  awk -F= -v key="${key}" '$1 == key { sub(/^[^=]*=/, ""); print; exit }' \
    "${UPGRADE_STATE_FILE}"
}

state_set() {
  local key="$1"
  local value="$2"
  local candidate="${UPGRADE_STATE_FILE}.candidate"
  [[ "${key}" =~ ^[a-z_]+$ ]] || fatal "invalid upgrade-state key: ${key}"
  [[ "${value}" != *$'\n'* && "${value}" != *$'\r'* ]] \
    || fatal "upgrade-state values must be single-line"

  if [ -f "${UPGRADE_STATE_FILE}" ]; then
    awk -F= -v key="${key}" '$1 != key { print }' \
      "${UPGRADE_STATE_FILE}" > "${candidate}"
  else
    : > "${candidate}"
  fi
  printf '%s=%s\n' "${key}" "${value}" >> "${candidate}"
  chmod 600 "${candidate}"
  mv "${candidate}" "${UPGRADE_STATE_FILE}"
}

initialize_upgrade_state() {
  local candidate_image="$1"
  local old_master_image="$2"
  local active_slot="$3"
  local active_image="$4"
  local fallback_slot="$5"
  local fallback_image="$6"
  local backup_file="$7"
  local report_file="$8"
  local candidate="${UPGRADE_STATE_FILE}.candidate"

  cat > "${candidate}" <<EOF
version=1
phase=preflight-complete
candidate_image=${candidate_image}
old_master_image=${old_master_image}
original_active_slot=${active_slot}
original_active_image=${active_image}
original_fallback_slot=${fallback_slot}
original_fallback_image=${fallback_image}
backup_file=${backup_file}
preflight_report=${report_file}
candidate_slot=
caddy_backup=
external_probe_file=
master_replaced=false
switched_at_epoch=
started_at=
completed_at=
EOF
  chmod 600 "${candidate}"
  mv "${candidate}" "${UPGRADE_STATE_FILE}"
}

require_upgrade_phase() {
  local phase
  local expected
  [ -f "${UPGRADE_STATE_FILE}" ] || fatal "${UPGRADE_STATE_FILE} is missing"
  phase="$(state_get phase)"
  for expected in "$@"; do
    if [ "${phase}" = "${expected}" ]; then
      return
    fi
  done
  fatal "upgrade phase is ${phase:-unknown}; expected: $*"
}

latest_backup() {
  find "${BACKUP_DIR}" -maxdepth 1 -type f -name '*.sql.gz' -printf '%T@ %p\n' \
    | sort -nr \
    | awk 'NR == 1 { sub(/^[^ ]+ /, ""); newest = $0 } END { print newest }'
}

create_verified_backup() {
  local backup_file="${UPGRADE_BACKUP_FILE:-}"
  if [ -z "${backup_file}" ]; then
    [ -x "${DEPLOY_PATH}/backup-db.sh" ] \
      || fatal "${DEPLOY_PATH}/backup-db.sh is required for an upgrade"
    "${DEPLOY_PATH}/backup-db.sh" >&2
    backup_file="$(latest_backup)"
  fi
  [ -n "${backup_file}" ] || fatal "no PostgreSQL backup was found in ${BACKUP_DIR}"
  [ -f "${backup_file}" ] || fatal "backup file is missing: ${backup_file}"
  gzip -t "${backup_file}" || fatal "backup integrity check failed: ${backup_file}"
  printf '%s\n' "${backup_file}"
}

container_running() {
  [ "$(docker inspect -f '{{.State.Running}}' "$1" 2>/dev/null || true)" = "true" ]
}

connection_count() {
  local container="$1"
  local pid
  if ! container_running "${container}"; then
    printf '0\n'
    return
  fi
  pid="$(docker inspect -f '{{.State.Pid}}' "${container}")"
  nsenter -t "${pid}" -n ss -Htn state established '( sport = :3000 )' | wc -l
}

wait_healthy() {
  local container="$1"
  local attempt
  local status
  for attempt in $(seq 1 90); do
    status="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "${container}" 2>/dev/null || true)"
    if [ "${status}" = "healthy" ]; then
      log "${container} is healthy"
      return
    fi
    if [ "${status}" = "unhealthy" ]; then
      docker logs --tail 200 "${container}" >&2 || true
      fatal "${container} became unhealthy"
    fi
    sleep 2
  done
  docker logs --tail 200 "${container}" >&2 || true
  fatal "timed out waiting for ${container} health"
}

slot_ip() {
  docker inspect -f '{{with index .NetworkSettings.Networks "new-api-net"}}{{.IPAddress}}{{end}}' "$1"
}

smoke_slot() {
  local slot="$1"
  local require_ready="${2:-false}"
  local service
  local ip
  local response
  service="$(service_for_slot "${slot}")"
  ip="$(slot_ip "${service}")"
  [ -n "${ip}" ] || fatal "could not resolve ${service} IP"

  response="$(curl --fail --silent --show-error --max-time 10 "http://${ip}:3000/api/status")"
  grep -q '"success"[[:space:]]*:[[:space:]]*true' <<< "${response}" \
    || fatal "${service} status smoke test failed"

  docker exec "${CADDY_CONTAINER}" wget -q -O /dev/null \
    "http://${service}:3000/api/status" \
    || fatal "Caddy cannot reach ${service}"

  if [ "${require_ready}" = "true" ]; then
    response="$(curl --fail --silent --show-error --max-time 10 "http://${ip}:3000/readyz")"
    grep -q '"ready"[[:space:]]*:[[:space:]]*true' <<< "${response}" \
      || fatal "${service} readiness smoke test failed"
    docker exec "${CADDY_CONTAINER}" wget -q -O /dev/null \
      "http://${service}:3000/readyz" \
      || fatal "Caddy cannot reach ${service} readiness endpoint"
  fi

  if [ -n "${SMOKE_TOKEN}" ]; then
    response="$(curl --fail --silent --show-error --max-time 15 \
      -H "Authorization: Bearer ${SMOKE_TOKEN}" \
      "http://${ip}:3000/v1/models")"
    grep -q '"data"' <<< "${response}" \
      || fatal "${service} authenticated model-list smoke test failed"
  else
    log "SMOKE_TOKEN is unset; skipping authenticated model-list smoke test"
  fi

  if [ -n "${SMOKE_TOKEN}" ] && [ -n "${SMOKE_MODEL}" ]; then
    local attempt
    local relay_ok
    local request_body
    request_body="{\"model\":\"${SMOKE_MODEL}\",\"messages\":[{\"role\":\"user\",\"content\":\"Reply with OK.\"}],\"max_tokens\":8,\"stream\":false}"
    relay_ok=0
    for attempt in $(seq 1 5); do
      if response="$(curl --fail --silent --show-error --max-time 90 \
        -H "Authorization: Bearer ${SMOKE_TOKEN}" \
        -H 'Content-Type: application/json' \
        --data "${request_body}" \
        "http://${ip}:3000/v1/chat/completions")" \
        && grep -q '"choices"' <<< "${response}"; then
        relay_ok=1
        break
      fi
      log "${service} non-stream relay smoke attempt ${attempt} failed"
      sleep 3
    done
    [ "${relay_ok}" -eq 1 ] || fatal "${service} non-stream relay smoke test failed"

    request_body="{\"model\":\"${SMOKE_MODEL}\",\"messages\":[{\"role\":\"user\",\"content\":\"Reply with OK.\"}],\"max_tokens\":8,\"stream\":true}"
    relay_ok=0
    for attempt in $(seq 1 5); do
      if response="$(curl --fail --silent --show-error --no-buffer --max-time 90 \
        -H "Authorization: Bearer ${SMOKE_TOKEN}" \
        -H 'Content-Type: application/json' \
        --data "${request_body}" \
        "http://${ip}:3000/v1/chat/completions")" \
        && grep -q '^data:' <<< "${response}" \
        && grep -qF 'data: [DONE]' <<< "${response}"; then
        relay_ok=1
        break
      fi
      log "${service} stream relay smoke attempt ${attempt} failed"
      sleep 3
    done
    [ "${relay_ok}" -eq 1 ] || fatal "${service} stream relay smoke test failed"
  else
    log "SMOKE_MODEL or SMOKE_TOKEN is unset; skipping live relay smoke tests"
  fi
}

render_caddy_candidate() {
  local primary="$1"
  local fallback="$2"
  local candidate="$3"
  local health_path="$4"
  awk -v primary="${primary}" -v fallback="${fallback}" -v health_path="${health_path}" '
    /# BEGIN NEW_API_UPSTREAM/ {
      print
      print "        reverse_proxy " primary ":3000 " fallback ":3000 {"
      print "            lb_policy first"
      print "            health_uri " health_path
      print "            health_interval 5s"
      print "            health_timeout 2s"
      print "            health_fails 2"
      print "            health_passes 2"
      print "            fail_duration 30s"
      print "            max_fails 1"
      print "        }"
      skip = 1
      next
    }
    /# END NEW_API_UPSTREAM/ {
      skip = 0
      print
      next
    }
    !skip { print }
  ' "${CADDYFILE}" > "${candidate}"
}

replace_file_atomically() {
  local source="$1"
  local destination="$2"
  local candidate="${destination}.replace.$$"

  [ -f "${source}" ] || fatal "replacement source is missing: ${source}"
  [ -f "${destination}" ] || fatal "replacement destination is missing: ${destination}"
  rm -f "${candidate}"
  cp -a "${destination}" "${candidate}"
  cp "${source}" "${candidate}"
  mv -f "${candidate}" "${destination}"
}

caddy_has_upstream_order() {
  local primary_service="$1"
  local fallback_service="$2"
  local active_config

  active_config="$(docker exec "${CADDY_CONTAINER}" wget -q -O - http://127.0.0.1:2019/config/)" \
    || return 1
  grep -q "\"dial\":\"${primary_service}:3000\".*\"dial\":\"${fallback_service}:3000\"" \
    <<< "${active_config}"
}

switch_caddy() {
  local target_slot="$1"
  local previous_slot="$2"
  local health_path="${3:-/api/status}"
  local target_service
  local previous_service
  local stamp
  local candidate
  local backup
  target_service="$(service_for_slot "${target_slot}")"
  previous_service="$(service_for_slot "${previous_slot}")"
  stamp="$(date +%F_%H%M%S)"
  candidate="${CADDYFILE}.candidate.${stamp}"
  backup="${CADDYFILE}.backup.${stamp}"

  grep -q '# BEGIN NEW_API_UPSTREAM' "${CADDYFILE}" || fatal "Caddyfile upstream marker is missing"
  grep -q '# END NEW_API_UPSTREAM' "${CADDYFILE}" || fatal "Caddyfile upstream end marker is missing"

  render_caddy_candidate "${target_service}" "${previous_service}" "${candidate}" "${health_path}"
  docker cp "${candidate}" "${CADDY_CONTAINER}:/tmp/Caddyfile.candidate"
  docker exec "${CADDY_CONTAINER}" caddy validate \
    --config /tmp/Caddyfile.candidate \
    --adapter caddyfile >/dev/null

  cp -a "${CADDYFILE}" "${backup}"
  if [ "${CAPTURE_CADDY_BACKUP_TO_STATE}" = "true" ]; then
    state_set caddy_backup "${backup}"
  fi
  replace_file_atomically "${candidate}" "${CADDYFILE}"
  if ! docker exec "${CADDY_CONTAINER}" caddy reload \
    --config /tmp/Caddyfile.candidate \
    --adapter caddyfile; then
    replace_file_atomically "${backup}" "${CADDYFILE}"
    docker cp "${backup}" "${CADDY_CONTAINER}:/tmp/Caddyfile.rollback"
    docker exec "${CADDY_CONTAINER}" caddy reload \
      --config /tmp/Caddyfile.rollback \
      --adapter caddyfile
    fatal "Caddy reload failed; previous config restored"
  fi

  if ! caddy_has_upstream_order "${target_service}" "${previous_service}"; then
    replace_file_atomically "${backup}" "${CADDYFILE}"
    docker cp "${backup}" "${CADDY_CONTAINER}:/tmp/Caddyfile.rollback"
    docker exec "${CADDY_CONTAINER}" caddy reload \
      --config /tmp/Caddyfile.rollback \
      --adapter caddyfile
    fatal "Caddy admin config did not contain the requested upstreams; previous config restored"
  fi

  if ! curl --fail --silent --show-error --max-time 15 "${PUBLIC_STATUS_URL}" >/dev/null; then
    replace_file_atomically "${backup}" "${CADDYFILE}"
    docker cp "${backup}" "${CADDY_CONTAINER}:/tmp/Caddyfile.rollback"
    docker exec "${CADDY_CONTAINER}" caddy reload \
      --config /tmp/Caddyfile.rollback \
      --adapter caddyfile
    fatal "public status check failed; previous Caddy config restored"
  fi
}

caddy_health_path_for_slots() {
  local primary="$1"
  local fallback="$2"
  if [ "$(slot_health_path "${primary}")" = "/readyz" ] \
    && [ "$(slot_health_path "${fallback}")" = "/readyz" ]; then
    printf '/readyz\n'
  else
    printf '/api/status\n'
  fi
}

restore_caddy_backup() {
  local backup="$1"
  [ -f "${backup}" ] || fatal "Caddy backup is missing: ${backup}"
  docker cp "${backup}" "${CADDY_CONTAINER}:/tmp/Caddyfile.rollback"
  docker exec "${CADDY_CONTAINER}" caddy validate \
    --config /tmp/Caddyfile.rollback \
    --adapter caddyfile >/dev/null
  replace_file_atomically "${backup}" "${CADDYFILE}"
  docker exec "${CADDY_CONTAINER}" caddy reload \
    --config /tmp/Caddyfile.rollback \
    --adapter caddyfile >/dev/null
  curl --fail --silent --show-error --max-time 15 "${PUBLIC_STATUS_URL}" >/dev/null \
    || fatal "public status check failed after Caddy rollback"
}

switch_slot() {
  local target_slot="$1"
  local previous_slot
  local target_service
  local previous_service
  validate_slot "${target_slot}"
  previous_slot="$(read_active_slot)"
  target_service="$(service_for_slot "${target_slot}")"
  previous_service="$(service_for_slot "${previous_slot}")"

  [ "${target_slot}" != "${previous_slot}" ] || fatal "${target_slot} is already active"
  container_running "${previous_service}" || fatal "active service ${previous_service} is not running"
  container_running "${target_service}" || fatal "${target_service} is not running"
  wait_healthy "${target_service}"
  smoke_slot "${target_slot}"
  switch_caddy \
    "${target_slot}" \
    "${previous_slot}" \
    "$(caddy_health_path_for_slots "${target_slot}" "${previous_slot}")"
  write_active_slot "${target_slot}"
  record_switch "${target_slot}" "${previous_slot}"
  log "active slot changed from ${previous_slot} to ${target_slot}"
}

deploy_inactive() {
  local image_ref="$1"
  local active_slot
  local inactive_slot
  local inactive_service
  local active_service
  local connections

  validate_image_ref "${image_ref}"

  active_slot="$(read_active_slot)"
  inactive_slot="$(other_slot "${active_slot}")"
  inactive_service="$(service_for_slot "${inactive_slot}")"
  active_service="$(service_for_slot "${active_slot}")"
  container_running "${active_service}" || fatal "active service ${active_service} is not running"
  connections="$(connection_count "${inactive_service}")"
  [ "${connections}" -eq 0 ] || fatal "${inactive_service} still has ${connections} established connection(s)"

  if [ -x "${DEPLOY_PATH}/backup-db.sh" ]; then
    "${DEPLOY_PATH}/backup-db.sh"
  fi

  registry_login
  update_slot_image "${inactive_slot}" "${image_ref}"
  compose pull "${inactive_service}"
  compose up -d --no-deps --force-recreate "${inactive_service}"
  wait_healthy "${inactive_service}"
  smoke_slot "${inactive_slot}"
  switch_caddy \
    "${inactive_slot}" \
    "${active_slot}" \
    "$(caddy_health_path_for_slots "${inactive_slot}" "${active_slot}")"
  write_active_slot "${inactive_slot}"
  record_switch "${inactive_slot}" "${active_slot}"
  log "deployed ${image_ref} to ${inactive_slot}; ${active_slot} remains running"
}

container_image() {
  docker inspect -f '{{.Config.Image}}' "$1"
}

run_production_migration() {
  local image_ref="$1"
  env \
    MASTER_IMAGE="${image_ref}" \
    MASTER_HEALTH_PATH=/readyz \
    docker compose \
      --env-file "${BASE_ENV_FILE}" \
      --env-file "${SLOT_ENV_FILE}" \
      -f "${COMPOSE_FILE}" \
      run --rm --no-deps \
      -e NODE_TYPE=master \
      -e NODE_NAME=new-api-upgrade-migration \
      -e BATCH_UPDATE_ENABLED=false \
      new-api-master \
      --migrate-only --log-dir /app/logs
}

replace_master_image() {
  local image_ref="$1"
  local health_path="$2"
  update_env_value MASTER_IMAGE "${image_ref}"
  update_env_value MASTER_HEALTH_PATH "${health_path}"
  compose pull new-api-master
  compose stop -t 960 new-api-master
  [ "$(docker inspect -f '{{.State.Running}}' new-api-master 2>/dev/null || true)" != "true" ] \
    || fatal "old master did not stop"
  compose up -d --no-deps --force-recreate new-api-master
  wait_healthy new-api-master
}

start_external_probe() {
  local observer="${DEPLOY_PATH}/observe-public.sh"
  [ -x "${observer}" ] || fatal "${observer} is required for start-upgrade"

  install -d -m 700 "${DEPLOY_PATH}/observations"
  EXTERNAL_PROBE_FILE="${DEPLOY_PATH}/observations/upgrade-$(date +%F_%H%M%S).csv"
  bash "${observer}" \
    "${UPGRADE_EXTERNAL_PROBE_SECONDS}" \
    "${UPGRADE_OBSERVE_INTERVAL}" \
    "${PUBLIC_STATUS_URL}" \
    "${EXTERNAL_PROBE_FILE}" &
  EXTERNAL_PROBE_PID="$!"
  state_set external_probe_file "${EXTERNAL_PROBE_FILE}"

  for _ in $(seq 1 40); do
    if awk 'NR > 1 { found = 1 } END { exit !found }' "${EXTERNAL_PROBE_FILE}" 2>/dev/null \
      && kill -0 "${EXTERNAL_PROBE_PID}" 2>/dev/null; then
      log "started continuous public probe; pid=${EXTERNAL_PROBE_PID} output=${EXTERNAL_PROBE_FILE}"
      return
    fi
    sleep 0.25
  done
  fatal "continuous public probe did not start"
}

external_probe_failure_count() {
  [ -f "${EXTERNAL_PROBE_FILE}" ] || {
    printf '1\n'
    return
  }
  awk -F, '
    NR > 1 && ($2 !~ /^[0-9]+$/ || $2 < 200 || $2 >= 300 || $4 != 0) { failures++ }
    END { print failures + 0 }
  ' "${EXTERNAL_PROBE_FILE}"
}

assert_external_probe_clean() {
  local failures
  [ -n "${EXTERNAL_PROBE_PID}" ] || return 1
  kill -0 "${EXTERNAL_PROBE_PID}" 2>/dev/null || return 1
  failures="$(external_probe_failure_count)"
  if [ "${failures}" -ne 0 ]; then
    log "continuous public probe recorded ${failures} failure(s); inspect ${EXTERNAL_PROBE_FILE}"
    return 1
  fi
}

cleanup_background_tasks() {
  if [ -n "${DRAIN_SSE_PID}" ] && kill -0 "${DRAIN_SSE_PID}" 2>/dev/null; then
    kill -TERM "${DRAIN_SSE_PID}" 2>/dev/null || true
    wait "${DRAIN_SSE_PID}" 2>/dev/null || true
  fi
  if [ -n "${EXTERNAL_PROBE_PID}" ] && kill -0 "${EXTERNAL_PROBE_PID}" 2>/dev/null; then
    kill -TERM "${EXTERNAL_PROBE_PID}" 2>/dev/null || true
    wait "${EXTERNAL_PROBE_PID}" 2>/dev/null || true
  fi
}

start_drain_sse() {
  local slot="$1"
  local service
  local attempt
  local connections
  local request_body
  [ -n "${SMOKE_TOKEN}" ] || return 1
  [ -n "${SMOKE_MODEL}" ] || return 1

  service="$(service_for_slot "${slot}")"
  install -d -m 700 "${DEPLOY_PATH}/upgrade-sse"
  DRAIN_SSE_FILE="${DEPLOY_PATH}/upgrade-sse/$(date +%F_%H%M%S)-${slot}.log"
  request_body="{\"model\":\"${SMOKE_MODEL}\",\"messages\":[{\"role\":\"user\",\"content\":\"Count from 1 to 1000, one number per line, with no other text.\"}],\"max_tokens\":1024,\"stream\":true}"

  curl --fail --silent --show-error --no-buffer --max-time 900 \
    -H "Authorization: Bearer ${SMOKE_TOKEN}" \
    -H 'Content-Type: application/json' \
    --data "${request_body}" \
    "${PUBLIC_CHAT_URL}" > "${DRAIN_SSE_FILE}" 2>&1 &
  DRAIN_SSE_PID="$!"

  for attempt in $(seq 1 40); do
    if ! kill -0 "${DRAIN_SSE_PID}" 2>/dev/null; then
      wait "${DRAIN_SSE_PID}" 2>/dev/null || true
      log "public drain SSE completed before the Caddy reload; inspect ${DRAIN_SSE_FILE}"
      return 1
    fi
    if grep -qF 'data: [DONE]' "${DRAIN_SSE_FILE}" 2>/dev/null; then
      log "public drain SSE reached [DONE] before the Caddy reload"
      return 1
    fi
    connections="$(connection_count "${service}")"
    if grep -q '^data:' "${DRAIN_SSE_FILE}" 2>/dev/null && [ "${connections}" -gt 0 ]; then
      log "established public drain SSE through Caddy on ${service}; pid=${DRAIN_SSE_PID}"
      return
    fi
    sleep 0.25
  done
  log "public drain SSE did not produce an initial data frame on ${service}"
  return 1
}

verify_drain_sse() {
  [ -n "${DRAIN_SSE_PID}" ] || return 1
  if ! wait "${DRAIN_SSE_PID}"; then
    log "drain SSE failed; inspect ${DRAIN_SSE_FILE}"
    return 1
  fi
  grep -q '^data:' "${DRAIN_SSE_FILE}" \
    || return 1
  grep -qF 'data: [DONE]' "${DRAIN_SSE_FILE}" \
    || return 1
  log "drain SSE completed with [DONE] after the Caddy reload"
}

wait_zero_connections() {
  local container="$1"
  local required_seconds="$2"
  local zero_since=0
  local now
  local count

  while :; do
    count="$(connection_count "${container}")"
    now="$(date +%s)"
    if [ "${count}" -eq 0 ]; then
      if [ "${zero_since}" -eq 0 ]; then
        zero_since="${now}"
      fi
      if [ $((now - zero_since)) -ge "${required_seconds}" ]; then
        log "${container} held zero established connections for ${required_seconds} seconds"
        return
      fi
    else
      zero_since=0
      log "${container} still has ${count} established connection(s); zero timer reset"
    fi
    sleep 5
  done
}

observe_started_upgrade() {
  local candidate_slot="$1"
  local candidate_service
  local deadline
  local failures=0
  local baseline_restarts
  local health
  local current_restarts
  local response
  candidate_service="$(service_for_slot "${candidate_slot}")"
  baseline_restarts="$(docker inspect -f '{{.RestartCount}}' "${candidate_service}")"
  deadline=$(( $(date +%s) + UPGRADE_OBSERVE_SECONDS ))

  while [ "$(date +%s)" -lt "${deadline}" ]; do
    health="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "${candidate_service}" 2>/dev/null || true)"
    current_restarts="$(docker inspect -f '{{.RestartCount}}' "${candidate_service}" 2>/dev/null || printf '%s' -1)"

    if [ "${health}" = "healthy" ] \
      && [ "${current_restarts}" = "${baseline_restarts}" ] \
      && curl --fail --silent --show-error --max-time 15 "${PUBLIC_STATUS_URL}" >/dev/null \
      && response="$(curl --fail --silent --show-error --max-time 20 \
        -H "Authorization: Bearer ${SMOKE_TOKEN}" \
        "${PUBLIC_MODELS_URL}")" \
      && grep -q '"data"' <<< "${response}"; then
      failures=0
    else
      failures=$((failures + 1))
      log "upgrade observation failure ${failures}/2 health=${health} restarts=${current_restarts}"
      if [ "${failures}" -ge 2 ]; then
        return 1
      fi
    fi
    sleep "${UPGRADE_OBSERVE_INTERVAL}"
  done
  return 0
}

rollback_upgrade_internal() {
  local phase
  local original_slot
  local old_master_image
  local caddy_backup
  local master_replaced
  phase="$(state_get phase)"
  original_slot="$(state_get original_active_slot)"
  old_master_image="$(state_get old_master_image)"
  caddy_backup="$(state_get caddy_backup)"
  master_replaced="$(state_get master_replaced)"

  if [ -n "${caddy_backup}" ] && [ "${phase}" != "preflight-complete" ]; then
    restore_caddy_backup "${caddy_backup}"
    write_active_slot "${original_slot}"
    log "Caddy traffic restored to original slot ${original_slot}"
  fi

  if [ "${master_replaced}" = "true" ] && [ -n "${old_master_image}" ]; then
    replace_master_image "${old_master_image}" /api/status
    log "master restored to ${old_master_image}"
  fi

  state_set phase rolled-back
  state_set completed_at "$(date -Ins)"
  log "upgrade rolled back; database schema and candidate logs were preserved"
}

preflight_upgrade() {
  local image_ref="$1"
  local existing_phase
  local active_slot
  local fallback_slot
  local active_service
  local fallback_service
  local active_image
  local fallback_image
  local old_master_image
  local backup_file
  local report_file

  validate_image_ref "${image_ref}"
  if [ -f "${UPGRADE_STATE_FILE}" ]; then
    existing_phase="$(state_get phase)"
    case "${existing_phase}" in
      finalized|rolled-back) ;;
      *) fatal "an upgrade is already in phase ${existing_phase:-unknown}" ;;
    esac
  fi

  active_slot="$(read_active_slot)"
  fallback_slot="$(other_slot "${active_slot}")"
  active_service="$(service_for_slot "${active_slot}")"
  fallback_service="$(service_for_slot "${fallback_slot}")"
  container_running "${active_service}" || fatal "${active_service} is not running"
  container_running "${fallback_service}" || fatal "${fallback_service} is not running"
  container_running new-api-master || fatal "new-api-master is not running"
  active_image="$(container_image "${active_service}")"
  fallback_image="$(container_image "${fallback_service}")"
  old_master_image="$(container_image new-api-master)"
  backup_file="$(create_verified_backup)"
  report_file="${PREFLIGHT_REPORT_DIR}/$(date +%F_%H%M%S)-$(printf '%s' "${image_ref##*@sha256:}" | cut -c1-12).txt"

  registry_login
  docker pull "${image_ref}"
  PREFLIGHT_REPORT_FILE="${report_file}" \
    BASE_ENV_FILE="${BASE_ENV_FILE}" \
    bash "${DEPLOY_PATH}/upgrade-preflight.sh" \
      "${image_ref}" \
      "${old_master_image}" \
      "${active_image}" \
      "${backup_file}"

  [ "$(read_active_slot)" = "${active_slot}" ] \
    || fatal "active slot changed during preflight; rerun the audit"
  [ "$(container_image "${active_service}")" = "${active_image}" ] \
    || fatal "active image changed during preflight; rerun the audit"
  [ "$(container_image new-api-master)" = "${old_master_image}" ] \
    || fatal "master image changed during preflight; rerun the audit"

  initialize_upgrade_state \
    "${image_ref}" \
    "${old_master_image}" \
    "${active_slot}" \
    "${active_image}" \
    "${fallback_slot}" \
    "${fallback_image}" \
    "${backup_file}" \
    "${report_file}"
  log "upgrade preflight passed; report=${report_file}"
}

start_upgrade() {
  local image_ref="$1"
  local active_slot
  local inactive_slot
  local active_service
  local inactive_service
  local backup_file

  validate_image_ref "${image_ref}"
  require_upgrade_phase preflight-complete
  [ "$(state_get candidate_image)" = "${image_ref}" ] \
    || fatal "candidate image does not match upgrade-state"
  active_slot="$(read_active_slot)"
  [ "${active_slot}" = "$(state_get original_active_slot)" ] \
    || fatal "active slot changed after preflight"
  active_service="$(service_for_slot "${active_slot}")"
  [ "$(container_image "${active_service}")" = "$(state_get original_active_image)" ] \
    || fatal "active image changed after preflight"
  [ "$(container_image new-api-master)" = "$(state_get old_master_image)" ] \
    || fatal "master image changed after preflight"

  inactive_slot="$(other_slot "${active_slot}")"
  inactive_service="$(service_for_slot "${inactive_slot}")"
  [ "$(container_image "${inactive_service}")" = "$(state_get original_fallback_image)" ] \
    || fatal "fallback image changed after preflight"
  [ "$(connection_count "${inactive_service}")" -eq 0 ] \
    || fatal "${inactive_service} still has established connections"
  [ -n "${SMOKE_TOKEN}" ] || fatal "SMOKE_TOKEN is required for start-upgrade"
  [ -n "${SMOKE_MODEL}" ] || fatal "SMOKE_MODEL is required for start-upgrade"
  caddy_has_upstream_order "${active_service}" "${inactive_service}" \
    || fatal "Caddy runtime upstream order does not match active-slot"

  wait_healthy "${active_service}"
  smoke_slot "${active_slot}"
  start_external_probe
  assert_external_probe_clean || fatal "initial public probe failed before production changes"

  backup_file="$(create_verified_backup)"
  state_set backup_file "${backup_file}"
  state_set phase starting
  state_set started_at "$(date -Ins)"

  registry_login
  docker pull "${image_ref}"
  if ! run_production_migration "${image_ref}"; then
    state_set phase rolled-back
    state_set completed_at "$(date -Ins)"
    fatal "production migrate-only failed; database backup was not auto-restored"
  fi
  if ! assert_external_probe_clean; then
    rollback_upgrade_internal
    fatal "public probe failed during the production migration"
  fi

  state_set master_replaced true
  if ! (replace_master_image "${image_ref}" /readyz); then
    rollback_upgrade_internal
    fatal "candidate master failed; old master restored"
  fi
  if ! assert_external_probe_clean; then
    rollback_upgrade_internal
    fatal "public probe failed during the master replacement"
  fi

  if ! (
    [ "$(connection_count "${inactive_service}")" -eq 0 ] \
      || fatal "${inactive_service} received fallback traffic before rebuild"
    update_slot_image "${inactive_slot}" "${image_ref}"
    update_slot_health_path "${inactive_slot}" /readyz
    compose pull "${inactive_service}"
    compose up -d --no-deps --force-recreate "${inactive_service}"
    wait_healthy "${inactive_service}"
    smoke_slot "${inactive_slot}" true
  ); then
    rollback_upgrade_internal
    fatal "candidate slot failed before traffic switch"
  fi
  if ! assert_external_probe_clean; then
    rollback_upgrade_internal
    fatal "public probe failed while preparing the candidate slot"
  fi

  if ! start_drain_sse "${active_slot}"; then
    rollback_upgrade_internal
    fatal "could not establish a long-lived SSE on the old active slot"
  fi
  if ! (
    CAPTURE_CADDY_BACKUP_TO_STATE="true"
    switch_caddy "${inactive_slot}" "${active_slot}" /api/status
  ); then
    rollback_upgrade_internal
    fatal "Caddy traffic switch failed"
  fi
  write_active_slot "${inactive_slot}"
  record_switch "${inactive_slot}" "${active_slot}"
  state_set candidate_slot "${inactive_slot}"
  state_set switched_at_epoch "$(date +%s)"
  state_set phase switched

  if ! verify_drain_sse; then
    rollback_upgrade_internal
    fatal "old-slot drain SSE did not finish cleanly"
  fi
  if ! (smoke_slot "${inactive_slot}" true); then
    rollback_upgrade_internal
    fatal "candidate smoke test failed after traffic switch"
  fi
  if ! assert_external_probe_clean; then
    rollback_upgrade_internal
    fatal "public probe recorded a non-2xx response during the traffic switch"
  fi
  if ! observe_started_upgrade "${inactive_slot}"; then
    rollback_upgrade_internal
    fatal "two consecutive upgrade observation probes failed"
  fi
  if ! assert_external_probe_clean; then
    rollback_upgrade_internal
    fatal "continuous public probe was not clean during the observation window"
  fi

  state_set phase observing-complete
  log "candidate served traffic for ${UPGRADE_OBSERVE_SECONDS} seconds; old slot remains untouched"
}

finalize_upgrade() {
  local candidate_image
  local candidate_slot
  local old_slot
  local old_service
  local switched_at
  local elapsed
  local remaining

  require_upgrade_phase observing-complete
  candidate_image="$(state_get candidate_image)"
  candidate_slot="$(state_get candidate_slot)"
  validate_slot "${candidate_slot}"
  [ "$(read_active_slot)" = "${candidate_slot}" ] \
    || fatal "candidate slot is no longer active"
  [ "$(container_image new-api-master)" = "${candidate_image}" ] \
    || fatal "master is not running the candidate image"

  switched_at="$(state_get switched_at_epoch)"
  [ -n "${switched_at}" ] || fatal "upgrade-state has no switch timestamp"
  elapsed=$(( $(date +%s) - switched_at ))
  if [ "${elapsed}" -lt "${UPGRADE_RETAIN_SECONDS}" ]; then
    remaining=$((UPGRADE_RETAIN_SECONDS - elapsed))
    fatal "old slot retention window has ${remaining} seconds remaining"
  fi

  old_slot="$(other_slot "${candidate_slot}")"
  old_service="$(service_for_slot "${old_slot}")"
  wait_zero_connections "${old_service}" "${UPGRADE_ZERO_CONNECTION_SECONDS}"

  if ! (
    update_slot_image "${old_slot}" "${candidate_image}"
    update_slot_health_path "${old_slot}" /readyz
    compose pull "${old_service}"
    compose up -d --no-deps --force-recreate "${old_service}"
    wait_healthy "${old_service}"
    smoke_slot "${old_slot}" true
  ); then
    update_slot_image "${old_slot}" "$(state_get original_active_image)"
    update_slot_health_path "${old_slot}" /api/status
    compose up -d --no-deps --force-recreate "${old_service}" || true
    fatal "failed to rebuild the retained old slot; attempted to restore its old image"
  fi

  switch_caddy "${candidate_slot}" "${old_slot}" /readyz
  state_set phase finalized
  state_set completed_at "$(date -Ins)"
  log "upgrade finalized; both slots and master use ${candidate_image}, Caddy health uses /readyz"
}

rollback_upgrade() {
  require_upgrade_phase preflight-complete starting switched observing-complete
  rollback_upgrade_internal
}

show_status() {
  local active="unknown"
  if [ -s "${ACTIVE_SLOT_FILE}" ]; then
    active="$(tr -d '[:space:]' < "${ACTIVE_SLOT_FILE}")"
  fi
  printf 'active_slot=%s\n' "${active}"
  for slot in blue green; do
    local service
    service="$(service_for_slot "${slot}")"
    if container_running "${service}"; then
      printf '%s running=true health=%s connections=%s image=%s\n' \
        "${service}" \
        "$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "${service}")" \
        "$(connection_count "${service}")" \
        "$(docker inspect -f '{{.Config.Image}}' "${service}")"
    else
      printf '%s running=false\n' "${service}"
    fi
  done
  if [ -s "${DEPLOY_HISTORY_FILE}" ]; then
    printf 'last_switch=%s\n' "$(tail -n 1 "${DEPLOY_HISTORY_FILE}")"
  fi
  if [ -f "${UPGRADE_STATE_FILE}" ]; then
    printf 'upgrade_phase=%s\n' "$(state_get phase)"
    printf 'upgrade_candidate=%s\n' "$(state_get candidate_image)"
    printf 'upgrade_report=%s\n' "$(state_get preflight_report)"
    printf 'upgrade_public_probe=%s\n' "$(state_get external_probe_file)"
  else
    printf 'upgrade_phase=none\n'
  fi
}

main() {
  require_command awk
  require_command curl
  require_command docker
  require_command gzip
  require_command flock
  require_command nsenter
  require_command ss

  [ -f "${COMPOSE_FILE}" ] || fatal "${COMPOSE_FILE} is missing"
  [ -f "${BASE_ENV_FILE}" ] || fatal "${BASE_ENV_FILE} is missing"
  [ -f "${SLOT_ENV_FILE}" ] || fatal "${SLOT_ENV_FILE} is missing"
  [ -f "${CADDYFILE}" ] || fatal "${CADDYFILE} is missing"

  exec 9>"${LOCK_FILE}"
  flock -n 9 || fatal "another blue-green deployment is running"
  trap cleanup_background_tasks EXIT

  case "${1:-}" in
    status)
      show_status
      ;;
    switch)
      [ "$#" -eq 2 ] || fatal "switch requires a target slot"
      switch_slot "$2"
      ;;
    deploy)
      [ "$#" -eq 2 ] || fatal "deploy requires an immutable image reference"
      deploy_inactive "$2"
      ;;
    preflight-upgrade)
      [ "$#" -eq 2 ] || fatal "preflight-upgrade requires an immutable image reference"
      [ -f "${DEPLOY_PATH}/upgrade-preflight.sh" ] \
        || fatal "${DEPLOY_PATH}/upgrade-preflight.sh is missing"
      preflight_upgrade "$2"
      ;;
    start-upgrade)
      [ "$#" -eq 2 ] || fatal "start-upgrade requires an immutable image reference"
      start_upgrade "$2"
      ;;
    finalize-upgrade)
      [ "$#" -eq 1 ] || fatal "finalize-upgrade takes no image argument"
      finalize_upgrade
      ;;
    rollback-upgrade)
      [ "$#" -eq 1 ] || fatal "rollback-upgrade takes no image argument"
      rollback_upgrade
      ;;
    *)
      usage
      exit 2
      ;;
  esac
}

if [ "${BLUE_GREEN_LIB_ONLY:-false}" != "true" ]; then
  main "$@"
fi
