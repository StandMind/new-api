#!/usr/bin/env bash
set -u

DEPLOY_PATH="${DEPLOY_PATH:-/opt/new-api-stack}"
DURATION_SECONDS="${1:-86400}"
INTERVAL_SECONDS="${2:-5}"
PUBLIC_STATUS_URL="${3:-https://aivrae.com/api/status}"
START_STAMP="$(date +%F_%H%M%S)"
OUTPUT_FILE="${4:-${DEPLOY_PATH}/observations/public-${START_STAMP}.csv}"
SUMMARY_FILE="${OUTPUT_FILE}.summary"

case "${DURATION_SECONDS}" in
  ''|*[!0-9]*) printf 'duration must be a positive integer\n' >&2; exit 2 ;;
esac
case "${INTERVAL_SECONDS}" in
  ''|*[!0-9]*) printf 'interval must be a positive integer\n' >&2; exit 2 ;;
esac
[ "${DURATION_SECONDS}" -gt 0 ] || { printf 'duration must be greater than zero\n' >&2; exit 2; }
[ "${INTERVAL_SECONDS}" -gt 0 ] || { printf 'interval must be greater than zero\n' >&2; exit 2; }

umask 077
mkdir -p "$(dirname "${OUTPUT_FILE}")"
printf 'timestamp,http_code,latency_seconds,curl_exit\n' > "${OUTPUT_FILE}"

START_EPOCH="$(date +%s)"
DEADLINE_EPOCH="$((START_EPOCH + DURATION_SECONDS))"
TOTAL=0
FAILURES=0

write_summary() {
  local finished_at
  finished_at="$(date -Ins)"
  {
    printf 'started_epoch=%s\n' "${START_EPOCH}"
    printf 'finished_at=%s\n' "${finished_at}"
    printf 'url=%s\n' "${PUBLIC_STATUS_URL}"
    printf 'samples=%s\n' "${TOTAL}"
    printf 'failures=%s\n' "${FAILURES}"
    printf 'output=%s\n' "${OUTPUT_FILE}"
  } > "${SUMMARY_FILE}"
}
trap write_summary EXIT

while [ "$(date +%s)" -lt "${DEADLINE_EPOCH}" ]; do
  TIMESTAMP="$(date -Ins)"
  RESULT="$(curl --silent --show-error --max-time 10 \
    -o /dev/null \
    -w '%{http_code},%{time_total}' \
    "${PUBLIC_STATUS_URL}" 2>/dev/null)"
  CURL_EXIT=$?
  if [ -z "${RESULT}" ]; then
    RESULT='000,10'
  fi
  HTTP_CODE="${RESULT%%,*}"
  LATENCY="${RESULT#*,}"
  TOTAL=$((TOTAL + 1))
  if [ "${CURL_EXIT}" -ne 0 ] || [ "${HTTP_CODE}" -lt 200 ] || [ "${HTTP_CODE}" -ge 300 ]; then
    FAILURES=$((FAILURES + 1))
  fi
  printf '%s,%s,%s,%s\n' \
    "${TIMESTAMP}" \
    "${HTTP_CODE}" \
    "${LATENCY}" \
    "${CURL_EXIT}" \
    >> "${OUTPUT_FILE}"
  sleep "${INTERVAL_SECONDS}"
done
