#!/usr/bin/env bash
set -euo pipefail

# Quick install: 3x-ui + xray-node management layer.
# Usage: curl -fsSL .../install.sh | sudo bash
#    or: sudo ./scripts/install.sh

REPO_URL="${XRAY_NODE_REPO:-https://github.com/thethoughtcriminal/xray-node.git}"
REPO_BRANCH="${XRAY_NODE_BRANCH:-main}"
INSTALL_DIR="${XRAY_NODE_INSTALL_DIR:-/opt/xray-node}"
CONFIG_PATH="/etc/xray-node/config.yaml"
BIN_PATH="/usr/local/bin/xray-node"
SERVICE_PATH="/etc/systemd/system/xray-node.service"
APPLY_INBOUNDS="${XRAY_NODE_APPLY_INBOUNDS:-1}"

require_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    echo "Run as root: sudo $0" >&2
    exit 1
  fi
}

install_deps() {
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update -y
    apt-get install -y curl git ca-certificates golang-go
  elif command -v dnf >/dev/null 2>&1; then
    dnf install -y curl git ca-certificates golang
  else
    echo "Install curl, git, and Go manually, then re-run." >&2
    exit 1
  fi
}

install_3xui() {
  if command -v x-ui >/dev/null 2>&1; then
    echo "3x-ui already installed"
    return
  fi
  echo "Installing 3x-ui..."
  bash <(curl -Ls https://raw.githubusercontent.com/mhsanaei/3x-ui/master/install.sh)
}

clone_or_update_repo() {
  if [[ -d "${INSTALL_DIR}/.git" ]]; then
    echo "Updating xray-node source..."
    git -C "${INSTALL_DIR}" fetch origin "${REPO_BRANCH}"
    # Drop local/untracked files (e.g. go.sum from a failed build) before sync.
    git -C "${INSTALL_DIR}" clean -fd
    git -C "${INSTALL_DIR}" reset --hard "origin/${REPO_BRANCH}"
  else
    git clone --branch "${REPO_BRANCH}" "${REPO_URL}" "${INSTALL_DIR}"
  fi
}

build_binary() {
  echo "Building xray-node..."
  (
    cd "${INSTALL_DIR}"
    go mod download
    go build -o "${BIN_PATH}" ./cmd/xray-node
  )
}

write_config() {
  mkdir -p /etc/xray-node
  if [[ ! -f "${CONFIG_PATH}" ]]; then
    cp "${INSTALL_DIR}/configs/config.example.yaml" "${CONFIG_PATH}"
    NODE_API_KEY="$(openssl rand -hex 24)"
    sed -i "s/CHANGE_ME_NODE_API_KEY/${NODE_API_KEY}/" "${CONFIG_PATH}"
    chmod 600 "${CONFIG_PATH}"
    echo "Generated node API key in ${CONFIG_PATH}"
  else
    echo "Config exists: ${CONFIG_PATH}"
  fi
}

write_systemd() {
  cat >"${SERVICE_PATH}" <<EOF
[Unit]
Description=xray-node management API
After=network.target x-ui.service
Wants=x-ui.service

[Service]
Type=simple
ExecStart=${BIN_PATH} serve --config ${CONFIG_PATH}
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
  systemctl enable xray-node
  systemctl restart xray-node
}

apply_default_inbounds() {
  if [[ "${APPLY_INBOUNDS}" != "1" ]]; then
    return
  fi
  echo "Applying default inbound templates..."
  "${BIN_PATH}" inbound apply "${INSTALL_DIR}/configs/inbounds/vless-reality.yaml" --config "${CONFIG_PATH}" || true
  "${BIN_PATH}" inbound apply "${INSTALL_DIR}/configs/inbounds/hysteria2.yaml" --config "${CONFIG_PATH}" || true
  echo "Open 3x-ui panel and set Reality keys / TLS cert for hysteria2 if needed."
}

print_next_steps() {
  cat <<EOF

xray-node installed.

1) Create API token in 3x-ui:
   Panel Settings -> API -> Create token
   Put it into ${CONFIG_PATH} as panel.token

2) Restart API:
   systemctl restart xray-node

3) Examples:
   xray-node inbound list
   xray-node inbound apply ${INSTALL_DIR}/configs/inbounds/vless-reality.yaml
   xray-node client add --inbound vless-reality --email user@node
   xray-node client stats --inbound vless-reality --email user@node

4) HTTP API (local):
   curl -H "X-API-Key: <key from config>" http://127.0.0.1:9472/healthz

5) Uninstall everything:
   sudo xray-node uninstall
   # or: curl -fsSL .../scripts/uninstall.sh | sudo bash

EOF
}

main() {
  require_root
  install_deps
  install_3xui
  clone_or_update_repo
  build_binary
  write_config
  write_systemd
  apply_default_inbounds
  print_next_steps
}

main "$@"
