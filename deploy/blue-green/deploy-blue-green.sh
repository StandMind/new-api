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
SMOKE_TOKEN="${SMOKE_TOKEN:-}"
SMOKE_MODEL="${SMOKE_MODEL:-}"
DEPLOY_REGISTRY_USERNAME="${DEPLOY_REGISTRY_USERNAME:-}"
DEPLOY_REGISTRY_TOKEN="${DEPLOY_REGISTRY_TOKEN:-}"
DEPLOY_HISTORY_FILE="${DEPLOY_HISTORY_FILE:-${DEPLOY_PATH}/deployment-history.log}"

usage() {
  cat <<'EOF'
Usage:
  deploy-blue-green.sh status
  deploy-blue-green.sh switch <blue|green>
  deploy-blue-green.sh deploy <image@sha256:digest>

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

update_slot_image() {
  local slot="$1"
  local image_ref="$2"
  local key
  local candidate="${SLOT_ENV_FILE}.candidate"
  key="$(image_key_for_slot "${slot}")"

  awk -F= -v key="${key}" -v value="${image_ref}" '
    BEGIN { updated = 0 }
    $1 == key { print key "=" value; updated = 1; next }
    { print }
    END { if (!updated) print key "=" value }
  ' "${SLOT_ENV_FILE}" > "${candidate}"
  chmod 600 "${candidate}"
  mv "${candidate}" "${SLOT_ENV_FILE}"
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
    local request_body
    request_body="{\"model\":\"${SMOKE_MODEL}\",\"messages\":[{\"role\":\"user\",\"content\":\"Reply with OK.\"}],\"max_tokens\":8,\"stream\":false}"
    response="$(curl --fail --silent --show-error --max-time 90 \
      -H "Authorization: Bearer ${SMOKE_TOKEN}" \
      -H 'Content-Type: application/json' \
      --data "${request_body}" \
      "http://${ip}:3000/v1/chat/completions")"
    grep -q '"choices"' <<< "${response}" \
      || fatal "${service} non-stream relay smoke test failed"

    request_body="{\"model\":\"${SMOKE_MODEL}\",\"messages\":[{\"role\":\"user\",\"content\":\"Reply with OK.\"}],\"max_tokens\":8,\"stream\":true}"
    response="$(curl --fail --silent --show-error --no-buffer --max-time 90 \
      -H "Authorization: Bearer ${SMOKE_TOKEN}" \
      -H 'Content-Type: application/json' \
      --data "${request_body}" \
      "http://${ip}:3000/v1/chat/completions")"
    grep -q '^data:' <<< "${response}" \
      || fatal "${service} stream relay smoke test failed"
  else
    log "SMOKE_MODEL or SMOKE_TOKEN is unset; skipping live relay smoke tests"
  fi
}

render_caddy_candidate() {
  local primary="$1"
  local fallback="$2"
  local candidate="$3"
  awk -v primary="${primary}" -v fallback="${fallback}" '
    /# BEGIN NEW_API_UPSTREAM/ {
      print
      print "        reverse_proxy " primary ":3000 " fallback ":3000 {"
      print "            lb_policy first"
      print "            health_uri /api/status"
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

switch_caddy() {
  local target_slot="$1"
  local previous_slot="$2"
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

  render_caddy_candidate "${target_service}" "${previous_service}" "${candidate}"
  docker cp "${candidate}" "${CADDY_CONTAINER}:/tmp/Caddyfile.candidate"
  docker exec "${CADDY_CONTAINER}" caddy validate \
    --config /tmp/Caddyfile.candidate \
    --adapter caddyfile >/dev/null

  cp -a "${CADDYFILE}" "${backup}"
  cp "${candidate}" "${CADDYFILE}"
  if ! docker exec "${CADDY_CONTAINER}" caddy reload \
    --config /tmp/Caddyfile.candidate \
    --adapter caddyfile; then
    cp "${backup}" "${CADDYFILE}"
    docker cp "${backup}" "${CADDY_CONTAINER}:/tmp/Caddyfile.rollback"
    docker exec "${CADDY_CONTAINER}" caddy reload \
      --config /tmp/Caddyfile.rollback \
      --adapter caddyfile
    fatal "Caddy reload failed; previous config restored"
  fi

  local active_config
  active_config="$(docker exec "${CADDY_CONTAINER}" wget -q -O - http://127.0.0.1:2019/config/)"
  if ! grep -q "\"dial\":\"${target_service}:3000\".*\"dial\":\"${previous_service}:3000\"" \
    <<< "${active_config}"; then
    cp "${backup}" "${CADDYFILE}"
    docker cp "${backup}" "${CADDY_CONTAINER}:/tmp/Caddyfile.rollback"
    docker exec "${CADDY_CONTAINER}" caddy reload \
      --config /tmp/Caddyfile.rollback \
      --adapter caddyfile
    fatal "Caddy admin config did not contain the requested upstreams; previous config restored"
  fi

  if ! curl --fail --silent --show-error --max-time 15 "${PUBLIC_STATUS_URL}" >/dev/null; then
    cp "${backup}" "${CADDYFILE}"
    docker cp "${backup}" "${CADDY_CONTAINER}:/tmp/Caddyfile.rollback"
    docker exec "${CADDY_CONTAINER}" caddy reload \
      --config /tmp/Caddyfile.rollback \
      --adapter caddyfile
    fatal "public status check failed; previous Caddy config restored"
  fi
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
  switch_caddy "${target_slot}" "${previous_slot}"
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

  case "${image_ref}" in
    *@sha256:????????????????????????????????????????????????????????????????) ;;
    *) fatal "image must be an immutable image@sha256:digest reference" ;;
  esac

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
  switch_caddy "${inactive_slot}" "${active_slot}"
  write_active_slot "${inactive_slot}"
  record_switch "${inactive_slot}" "${active_slot}"
  log "deployed ${image_ref} to ${inactive_slot}; ${active_slot} remains running"
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
}

main() {
  require_command awk
  require_command curl
  require_command docker
  require_command flock
  require_command nsenter
  require_command ss

  [ -f "${COMPOSE_FILE}" ] || fatal "${COMPOSE_FILE} is missing"
  [ -f "${BASE_ENV_FILE}" ] || fatal "${BASE_ENV_FILE} is missing"
  [ -f "${SLOT_ENV_FILE}" ] || fatal "${SLOT_ENV_FILE} is missing"
  [ -f "${CADDYFILE}" ] || fatal "${CADDYFILE} is missing"

  exec 9>"${LOCK_FILE}"
  flock -n 9 || fatal "another blue-green deployment is running"

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
    *)
      usage
      exit 2
      ;;
  esac
}

main "$@"
