#!/usr/bin/env bash
set -euo pipefail

# Remove xray-node and (by default) 3x-ui installed by scripts/install.sh.
# Usage: curl -fsSL .../uninstall.sh | sudo bash
#    or: sudo ./scripts/uninstall.sh [--yes] [--keep-3xui]

INSTALL_DIR="${XRAY_NODE_INSTALL_DIR:-/opt/xray-node}"
CONFIG_DIR="/etc/xray-node"
BIN_PATH="/usr/local/bin/xray-node"
SERVICE_PATH="/etc/systemd/system/xray-node.service"

KEEP_3XUI=0
ASSUME_YES=0

usage() {
  cat <<EOF
Usage: $0 [options]

Options:
  -y, --yes       Skip confirmation
  --keep-3xui     Remove only xray-node, keep 3x-ui / Xray
  -h, --help      Show this help
EOF
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -y | --yes)
        ASSUME_YES=1
        shift
        ;;
      --keep-3xui)
        KEEP_3XUI=1
        shift
        ;;
      -h | --help)
        usage
        exit 0
        ;;
      *)
        echo "Unknown option: $1" >&2
        usage >&2
        exit 1
        ;;
    esac
  done
}

require_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    echo "Run as root: sudo $0" >&2
    exit 1
  fi
}

confirm() {
  if [[ "${ASSUME_YES}" -eq 1 ]]; then
    return 0
  fi
  echo "This will remove:"
  echo "  - xray-node service, binary, config, and ${INSTALL_DIR}"
  if [[ "${KEEP_3XUI}" -eq 0 ]]; then
    echo "  - 3x-ui panel, Xray, and related data"
  else
    echo "  - 3x-ui will be kept (--keep-3xui)"
  fi
  read -r -p "Continue? [y/N] " reply
  case "${reply}" in
    y | Y | yes | YES) ;;
    *)
      echo "Aborted."
      exit 0
      ;;
  esac
}

remove_xray_node() {
  echo "Stopping xray-node..."
  systemctl stop xray-node 2>/dev/null || true
  systemctl disable xray-node 2>/dev/null || true

  if [[ -f "${SERVICE_PATH}" ]]; then
    rm -f "${SERVICE_PATH}"
  fi
  systemctl daemon-reload
  systemctl reset-failed 2>/dev/null || true

  if [[ -f "${BIN_PATH}" ]]; then
    rm -f "${BIN_PATH}"
  fi
  if [[ -d "${CONFIG_DIR}" ]]; then
    rm -rf "${CONFIG_DIR}"
  fi
  if [[ -d "${INSTALL_DIR}" ]]; then
    rm -rf "${INSTALL_DIR}"
  fi
}

remove_3xui() {
  if command -v x-ui >/dev/null 2>&1; then
    echo "Uninstalling 3x-ui via x-ui uninstall..."
    if [[ "${ASSUME_YES}" -eq 1 ]]; then
      printf 'y\n' | x-ui uninstall || true
    else
      x-ui uninstall || true
    fi
    return
  fi

  echo "x-ui command not found, removing 3x-ui files manually..."
  systemctl stop x-ui 2>/dev/null || true
  systemctl disable x-ui 2>/dev/null || true
  rm -f /etc/systemd/system/x-ui.service /usr/lib/systemd/system/x-ui.service
  systemctl daemon-reload
  systemctl reset-failed 2>/dev/null || true
  rm -rf /etc/x-ui /usr/local/x-ui
  rm -f /usr/bin/x-ui /usr/local/bin/x-ui
}

print_done() {
  cat <<EOF

Uninstall complete.

Re-install:
  curl -fsSL https://raw.githubusercontent.com/thethoughtcriminal/xray-node/main/scripts/install.sh | sudo bash

EOF
}

main() {
  parse_args "$@"
  require_root
  confirm
  remove_xray_node
  if [[ "${KEEP_3XUI}" -eq 0 ]]; then
    remove_3xui
  fi
  print_done
}

main "$@"
