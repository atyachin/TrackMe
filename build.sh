#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${ROOT_DIR}"

if ! command -v go >/dev/null 2>&1; then
  echo "Error: go is not installed or not in PATH" >&2
  exit 1
fi

# Build output / config path
OUTPUT_BIN="${OUTPUT_BIN:-TrackMe}"
CONFIG_PATH="${CONFIG_PATH:-config.json}"

# Server settings
TLS_PORT="${TLS_PORT:-443}"
HTTP_PORT="${HTTP_PORT:-80}"
HOST="${HOST:-0.0.0.0}"
DEVICE="${DEVICE:-auto}"
CORS_KEY="${CORS_KEY:-X-CORS}"
LOG_FILE="${LOG_FILE:-/var/log/TrackMe.log}"

# Cert settings: either set CERT_FILE + KEY_FILE, or set DOMAIN to derive both.
DOMAIN="${DOMAIN:-tagmon.apibox.cloud}"
CERT_FILE="${CERT_FILE:-}"
KEY_FILE="${KEY_FILE:-}"

if [[ -z "${CERT_FILE}" || -z "${KEY_FILE}" ]]; then
  if [[ -z "${DOMAIN}" ]]; then
    echo "Error: set DOMAIN or set both CERT_FILE and KEY_FILE" >&2
    echo "Example: DOMAIN=tagmon.apibox.cloud ./build.sh" >&2
    exit 1
  fi
  CERT_FILE="/etc/letsencrypt/live/${DOMAIN}/fullchain.pem"
  KEY_FILE="/etc/letsencrypt/live/${DOMAIN}/privkey.pem"
fi

HTTP_REDIRECT="${HTTP_REDIRECT:-https://${DOMAIN}}"

ENABLE_QUIC="${ENABLE_QUIC:-true}"
ENABLE_QUIC="$(echo "${ENABLE_QUIC}" | tr '[:upper:]' '[:lower:]')"
if [[ "${ENABLE_QUIC}" != "true" && "${ENABLE_QUIC}" != "false" ]]; then
  echo "Error: ENABLE_QUIC must be true or false" >&2
  exit 1
fi

echo "Building binary: ${OUTPUT_BIN}"
go build -o "${OUTPUT_BIN}" ./cmd/main.go

if [[ -f "${CONFIG_PATH}" ]]; then
  cp "${CONFIG_PATH}" "${CONFIG_PATH}.bak.$(date +%Y%m%d%H%M%S)"
fi

echo "Writing config: ${CONFIG_PATH}"
cat > "${CONFIG_PATH}" <<EOF
{
  "tls_port": "${TLS_PORT}",
  "http_port": "${HTTP_PORT}",
  "cert_file": "${CERT_FILE}",
  "key_file": "${KEY_FILE}",
  "host": "${HOST}",
  "http_redirect": "${HTTP_REDIRECT}",
  "device": "${DEVICE}",
  "cors_key": "${CORS_KEY}",
  "log_file": "${LOG_FILE}",
  "enable_quic": ${ENABLE_QUIC}
}
EOF

echo "Done."
echo "Binary: ${ROOT_DIR}/${OUTPUT_BIN}"
echo "Config: ${ROOT_DIR}/${CONFIG_PATH}"
