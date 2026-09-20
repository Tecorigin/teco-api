#!/usr/bin/env bash

set -euo pipefail

# Required: NEW_API_KEY and MODEL_NAME.
# Optional:
#   API_BASE_URL=http://127.0.0.1:3000
#   RPM_LIMIT=2                 Configured RPM limit for this user/model rule.
#   TPM_LIMIT=100               Configured TPM limit for this user/model rule.
#   TPM_PROMPT_REPEAT=200       Increase when the prompt does not exceed TPM_LIMIT.
#   TPM_MAX_REQUESTS=20         Maximum repeated requests while accumulating TPM.
# Usage:
#   ./test_rate_limit.sh rpm     Test only RPM.
#   ./test_rate_limit.sh tpm     Test only TPM.
#   ./test_rate_limit.sh all     Test RPM then TPM (default).
#
# Before running, configure a user, group, or user+model RPM/TPM rule in
# System Settings -> Security & Limits -> RPM/TPM Limits. The script verifies
# that requests over each configured limit return HTTP 429.

: "${NEW_API_KEY:?Set NEW_API_KEY to an API key with the configured limit}"
: "${MODEL_NAME:?Set MODEL_NAME to the model covered by the configured limit}"

API_BASE_URL="http://10.10.30.8:3002"
RPM_LIMIT="${RPM_LIMIT:-2}"
TPM_LIMIT="${TPM_LIMIT:-100}"
TPM_PROMPT_REPEAT="${TPM_PROMPT_REPEAT:-200}"
TPM_MAX_REQUESTS="${TPM_MAX_REQUESTS:-20}"
ENDPOINT="${API_BASE_URL%/}/v1/chat/completions"
TEST_MODE="${1:-all}"

if [[ "$TEST_MODE" != "rpm" && "$TEST_MODE" != "tpm" && "$TEST_MODE" != "all" ]]; then
  printf 'Usage: %s [rpm|tpm|all]\n' "$0" >&2
  exit 2
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

request() {
  local payload_file="$1"
  local response_file="$2"

  # admin: sk-wJrAcW1MTlBgPHJkS3E1zomyFmqtefj6hVsDQflAtmpgvKn7
  # svip-zhaoling: sk-Oc4ypcGGKOFGrPQh8gIShptePnlulqsvnsnD0I6xLzstYVDv
  curl --silent --show-error --output "$response_file" --write-out '%{http_code}' \
    --connect-timeout 10 --max-time 120 \
    -H "Authorization: Bearer sk-Oc4ypcGGKOFGrPQh8gIShptePnlulqsvnsnD0I6xLzstYVDv" \
    -H 'Content-Type: application/json' \
    --data-binary "@$payload_file" \
    "$ENDPOINT"
}

assert_429() {
  local status="$1"
  local response_file="$2"
  local test_name="$3"

  if [[ "$status" != "429" ]]; then
    printf '%s failed: expected HTTP 429, got HTTP %s\n' "$test_name" "$status" >&2
    printf 'Response: ' >&2
    head -c 1000 "$response_file" >&2
    printf '\n' >&2
    exit 1
  fi
  printf '%s passed: received HTTP 429\n' "$test_name"
  printf 'Response: '
  cat "$response_file"
  printf '\n'
}

test_rpm() {
  cat > "$tmp_dir/rpm.json" <<EOF
{
  "model": "${MODEL_NAME}",
  "messages": [{"role": "user", "content": "Reply with exactly: ok"}],
  "max_tokens": 8,
  "temperature": 0
}
EOF

  printf 'Testing RPM=%s against %s\n' "$RPM_LIMIT" "$ENDPOINT"
  for index in $(seq 1 "$RPM_LIMIT"); do
    response_file="$tmp_dir/rpm-$index.json"
    request "$tmp_dir/rpm.json" "$response_file" > "$tmp_dir/rpm-$index.status" &
  done
  wait

  local rpm_status
  rpm_status="$(request "$tmp_dir/rpm.json" "$tmp_dir/rpm-over-limit.json")"
  assert_429 "$rpm_status" "$tmp_dir/rpm-over-limit.json" 'RPM limit'
}

test_tpm() {
  local long_prompt
  long_prompt="$(printf 'rate limit token test %.0s' $(seq 1 "$TPM_PROMPT_REPEAT"))"
  cat > "$tmp_dir/tpm.json" <<EOF
{
  "model": "${MODEL_NAME}",
  "messages": [{"role": "user", "content": "${long_prompt}"}],
  "max_tokens": 1,
  "temperature": 0
}
EOF

  printf 'Testing TPM=%s with a prompt repeated %s times per request\n' "$TPM_LIMIT" "$TPM_PROMPT_REPEAT"
  printf 'TPM is cumulative over 60 seconds; sending up to %s requests until HTTP 429.\n' "$TPM_MAX_REQUESTS"
  printf 'Set the configured RPM to 0 while testing TPM, otherwise RPM may trigger first.\n'

  local attempt response_file tpm_status
  for attempt in $(seq 1 "$TPM_MAX_REQUESTS"); do
    response_file="$tmp_dir/tpm-$attempt.json"
    tpm_status="$(request "$tmp_dir/tpm.json" "$response_file")"
    if [[ "$tpm_status" == "429" ]]; then
      printf 'TPM limit passed: request %s received HTTP 429\n' "$attempt"
      printf 'Response: '
      cat "$response_file"
      printf '\n'
      return
    fi
    if [[ "$tpm_status" != "200" ]]; then
      printf 'TPM limit failed: expected HTTP 200 or 429, got HTTP %s on request %s\n' "$tpm_status" "$attempt" >&2
      printf 'Response: ' >&2
      head -c 1000 "$response_file" >&2
      printf '\n' >&2
      exit 1
    fi
    printf 'TPM request %s accepted; continuing to accumulate window usage.\n' "$attempt"
  done

  printf 'TPM limit failed: no HTTP 429 after %s requests. Increase TPM_PROMPT_REPEAT or TPM_MAX_REQUESTS.\n' "$TPM_MAX_REQUESTS" >&2
  exit 1
}

case "$TEST_MODE" in
  rpm)
    test_rpm
    ;;
  tpm)
    test_tpm
    ;;
  all)
    test_rpm
    test_tpm
    ;;
esac

printf 'All rate-limit checks passed.\n'