#!/usr/bin/env bash
set -euo pipefail

# Register an installed xray-node with xray-master (self-enrollment).
# Usage (on the node VPS, after xray-node install):
#   curl -fsSL https://raw.githubusercontent.com/themnts/xray-node/main/scripts/join.sh | \
#     sudo MASTER_URL=https://master.example.com ENROLL_TOKEN=xxx NODE_NAME=nl-1 bash
#
# Or: xray-node join --master-url ... --token ... --name ...

MASTER_URL="${MASTER_URL:-}"
ENROLL_TOKEN="${ENROLL_TOKEN:-}"
NODE_NAME="${NODE_NAME:-}"
PUBLIC_HOST="${PUBLIC_HOST:-${NODE_PUBLIC_HOST:-}}"
MASTER_IP="${MASTER_IP:-}"

if [[ "${EUID}" -ne 0 ]]; then
  echo "Run as root: sudo $0" >&2
  exit 1
fi

if [[ -z "${MASTER_URL}" || -z "${ENROLL_TOKEN}" || -z "${NODE_NAME}" ]]; then
  echo "Required env: MASTER_URL ENROLL_TOKEN NODE_NAME" >&2
  exit 1
fi

if ! command -v xray-node >/dev/null 2>&1; then
  echo "xray-node not installed. Run install.sh first." >&2
  exit 1
fi

args=(join --master-url "${MASTER_URL}" --token "${ENROLL_TOKEN}" --name "${NODE_NAME}")
if [[ -n "${PUBLIC_HOST}" ]]; then
  args+=(--public-host "${PUBLIC_HOST}")
fi
if [[ -n "${MASTER_IP}" ]]; then
  args+=(--master-ip "${MASTER_IP}")
fi

exec xray-node "${args[@]}"
