#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEST_DIR="$(mktemp -d)"
trap 'rm -rf "${TEST_DIR}"' EXIT

export DEPLOY_PATH="${TEST_DIR}/stack"
export SLOT_ENV_FILE="${DEPLOY_PATH}/slots.env"
export ACTIVE_SLOT_FILE="${DEPLOY_PATH}/active-slot"
export UPGRADE_STATE_FILE="${DEPLOY_PATH}/upgrade-state"
export CADDYFILE="${TEST_DIR}/Caddyfile"
export BLUE_GREEN_LIB_ONLY=true

install -d "${DEPLOY_PATH}"
cat > "${SLOT_ENV_FILE}" <<'EOF'
BLUE_IMAGE=example.invalid/new-api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
GREEN_IMAGE=example.invalid/new-api@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
MASTER_IMAGE=example.invalid/new-api@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc
EOF
printf 'green\n' > "${ACTIVE_SLOT_FILE}"
cat > "${CADDYFILE}" <<'EOF'
aivrae.example {
    # BEGIN NEW_API_UPSTREAM
    reverse_proxy old-primary:3000 old-fallback:3000 {
        health_uri /api/status
    }
    # END NEW_API_UPSTREAM
}
EOF

# shellcheck source=deploy/blue-green/deploy-blue-green.sh
source "${ROOT_DIR}/deploy/blue-green/deploy-blue-green.sh"

VALID_IMAGE='example.invalid/new-api@sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd'
validate_image_ref "${VALID_IMAGE}"
if (validate_image_ref 'example.invalid/new-api:latest'); then
  printf 'mutable image unexpectedly passed validation\n' >&2
  exit 1
fi

update_slot_image blue "${VALID_IMAGE}"
[ "$(read_env_value BLUE_IMAGE)" = "${VALID_IMAGE}" ]
update_slot_health_path blue /readyz
[ "$(slot_health_path blue)" = "/readyz" ]
[ "$(slot_health_path green)" = "/api/status" ]

initialize_upgrade_state \
  "${VALID_IMAGE}" \
  "$(read_env_value MASTER_IMAGE)" \
  green \
  "$(read_env_value GREEN_IMAGE)" \
  blue \
  "$(read_env_value BLUE_IMAGE)" \
  "${DEPLOY_PATH}/backups/test.sql.gz" \
  "${DEPLOY_PATH}/preflight-reports/test.txt"
[ "$(state_get phase)" = "preflight-complete" ]
[ -z "$(state_get external_probe_file)" ]
state_set phase starting
[ "$(state_get phase)" = "starting" ]
state_set master_replaced true
[ "$(state_get master_replaced)" = "true" ]

CANDIDATE_CADDY="${TEST_DIR}/Caddyfile.candidate"
render_caddy_candidate new-api-blue new-api-green "${CANDIDATE_CADDY}" /readyz
grep -q 'reverse_proxy new-api-blue:3000 new-api-green:3000' "${CANDIDATE_CADDY}"
grep -q 'health_uri /readyz' "${CANDIDATE_CADDY}"
grep -q 'health_headers {' "${CANDIDATE_CADDY}"
grep -q 'Connection close' "${CANDIDATE_CADDY}"

ATOMIC_DESTINATION="${TEST_DIR}/atomic-destination"
ATOMIC_SOURCE="${TEST_DIR}/atomic-source"
printf 'old\n' > "${ATOMIC_DESTINATION}"
printf 'new\n' > "${ATOMIC_SOURCE}"
chmod 640 "${ATOMIC_DESTINATION}"
OLD_INODE="$(stat -c '%i' "${ATOMIC_DESTINATION}")"
replace_file_atomically "${ATOMIC_SOURCE}" "${ATOMIC_DESTINATION}"
[ "$(cat "${ATOMIC_DESTINATION}")" = 'new' ]
[ "$(stat -c '%a' "${ATOMIC_DESTINATION}")" = '640' ]
[ "$(stat -c '%i' "${ATOMIC_DESTINATION}")" != "${OLD_INODE}" ]

EXTERNAL_PROBE_FILE="${TEST_DIR}/public-probe.csv"
cat > "${EXTERNAL_PROBE_FILE}" <<'EOF'
timestamp,http_code,latency_seconds,curl_exit
2026-07-18T00:00:00+00:00,200,0.1,0
EOF
[ "$(external_probe_failure_count)" -eq 0 ]
printf '2026-07-18T00:00:05+00:00,503,0.1,0\n' >> "${EXTERNAL_PROBE_FILE}"
[ "$(external_probe_failure_count)" -eq 1 ]

EVENTS_FILE="${TEST_DIR}/rollback-events"
state_set phase switched
state_set caddy_backup "${TEST_DIR}/Caddyfile.backup"
state_set master_replaced true
restore_caddy_backup() {
  printf 'caddy\n' >> "${EVENTS_FILE}"
}
write_active_slot() {
  printf 'active:%s\n' "$1" >> "${EVENTS_FILE}"
}
replace_master_image() {
  printf 'master:%s:%s\n' "$1" "$2" >> "${EVENTS_FILE}"
}
rollback_upgrade_internal
EXPECTED_OLD_MASTER='example.invalid/new-api@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc'
[ "$(sed -n '1p' "${EVENTS_FILE}")" = 'caddy' ]
[ "$(sed -n '2p' "${EVENTS_FILE}")" = 'active:green' ]
[ "$(sed -n '3p' "${EVENTS_FILE}")" = "master:${EXPECTED_OLD_MASTER}:/api/status" ]
[ "$(state_get phase)" = 'rolled-back' ]

CONNECTION_COUNTS_FILE="${TEST_DIR}/connection-counts"
connection_count() {
  sed -n '1p' "${CONNECTION_COUNTS_FILE}"
  sed -i '1d' "${CONNECTION_COUNTS_FILE}"
}
sleep() {
  :
}

printf '1\n0\n0\n' > "${CONNECTION_COUNTS_FILE}"
wait_until_no_connections new-api-blue 2
[ ! -s "${CONNECTION_COUNTS_FILE}" ]

printf '1\n1\n' > "${CONNECTION_COUNTS_FILE}"
if (wait_until_no_connections new-api-blue 1); then
  printf 'connection drain timeout unexpectedly passed\n' >&2
  exit 1
fi

(
  export PREFLIGHT_LIB_ONLY=true
  export PREFLIGHT_REPORT_FILE="${TEST_DIR}/preflight-report.txt"
  # shellcheck source=deploy/blue-green/upgrade-preflight.sh
  source "${ROOT_DIR}/deploy/blue-green/upgrade-preflight.sh"

  POSTGRES_USER=test_user
  POSTGRES_DB=test_db
  MOCK_PID1=bash
  MOCK_QUERY_READY=false
  MOCK_QUERY_CALLS=0

  docker() {
    [ "$1" = exec ] || return 1
    case "$3" in
      sh)
        [ "${MOCK_PID1}" = postgres ]
        ;;
      psql)
        MOCK_QUERY_CALLS=$((MOCK_QUERY_CALLS + 1))
        [ "${MOCK_QUERY_READY}" = true ]
        ;;
      *)
        return 1
        ;;
    esac
  }

  if postgres_is_ready test-postgres; then
    printf 'temporary PostgreSQL entrypoint unexpectedly passed readiness\n' >&2
    exit 1
  fi
  [ "${MOCK_QUERY_CALLS}" -eq 0 ]

  MOCK_PID1=postgres
  if postgres_is_ready test-postgres; then
    printf 'PostgreSQL with a failing query unexpectedly passed readiness\n' >&2
    exit 1
  fi
  [ "${MOCK_QUERY_CALLS}" -eq 1 ]

  MOCK_QUERY_READY=true
  postgres_is_ready test-postgres
  [ "${MOCK_QUERY_CALLS}" -eq 2 ]
)

printf 'blue-green deployment helper tests passed\n'
