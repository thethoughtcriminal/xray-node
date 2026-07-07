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

XUI_INSTALL_RESULT="/etc/x-ui/install-result.env"

configure_panel_from_xui() {
  local panel_url="" panel_token="" insecure_tls="true"

  if [[ -f "${XUI_INSTALL_RESULT}" ]]; then
    # shellcheck disable=SC1090
    source "${XUI_INSTALL_RESULT}"
    local port="${XUI_PANEL_PORT:-2053}"
    local base="${XUI_WEB_BASE_PATH:-}"
    base="${base#/}"
    base="${base%/}"
    local scheme="https"
    if [[ "${XUI_ACCESS_URL:-}" == http://* ]]; then
      scheme="http"
    fi
    panel_url="${scheme}://127.0.0.1:${port}"
    if [[ -n "${base}" ]]; then
      panel_url="${panel_url}/${base}"
    fi
    panel_token="${XUI_API_TOKEN:-}"
  fi

  if [[ -z "${panel_token}" ]] && command -v x-ui >/dev/null 2>&1; then
    panel_token="$(x-ui setting -getApiToken true 2>/dev/null | grep -Eo 'apiToken: .+' | awk '{print $2}' || true)"
  fi

  if [[ -z "${panel_url}" ]] && command -v x-ui >/dev/null 2>&1; then
    local settings port base_path
    settings="$(x-ui settings 2>/dev/null || true)"
    port="$(echo "${settings}" | grep -Eo 'port: [0-9]+' | awk '{print $2}' | head -1)"
    base_path="$(echo "${settings}" | grep -Eo 'webBasePath: /[^ ]+' | awk '{print $2}' | head -1)"
    base_path="${base_path#/}"
    base_path="${base_path%/}"
    if [[ -n "${port}" ]]; then
      if echo "${settings}" | grep -qiE 'not secure with SSL|without SSL|plain HTTP'; then
        panel_url="http://127.0.0.1:${port}"
      else
        panel_url="https://127.0.0.1:${port}"
      fi
      if [[ -n "${base_path}" ]]; then
        panel_url="${panel_url}/${base_path}"
      fi
    fi
  fi

  if [[ -z "${panel_url}" || -z "${panel_token}" ]]; then
    echo "Could not auto-configure panel; set panel.url and panel.token in ${CONFIG_PATH}"
    return 1
  fi

  local tmp
  tmp="$(mktemp)"
  awk -v url="${panel_url}" -v token="${panel_token}" -v insecure="${insecure_tls}" '
    /^  url:/ { print "  url: " url; next }
    /^  token:/ { print "  token: " token; next }
    /^  insecure_tls:/ { print "  insecure_tls: " insecure; next }
    { print }
  ' "${CONFIG_PATH}" >"${tmp}"
  mv "${tmp}" "${CONFIG_PATH}"
  chmod 600 "${CONFIG_PATH}"
  echo "Configured panel from 3x-ui (${panel_url})"
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
  if [[ -f "${XUI_INSTALL_RESULT}" ]] || grep -q 'CHANGE_ME_PANEL_API_TOKEN' "${CONFIG_PATH}"; then
    configure_panel_from_xui || true
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
  "${BIN_PATH}" inbound apply "${INSTALL_DIR}/configs/inbounds/vless-reality.yaml" --config "${CONFIG_PATH}" --non-interactive || true
  "${BIN_PATH}" inbound apply "${INSTALL_DIR}/configs/inbounds/hysteria2.yaml" --config "${CONFIG_PATH}" --non-interactive || true
  echo "Open 3x-ui panel and set Reality keys / TLS cert for hysteria2 if needed."
}

print_next_steps() {
  cat <<EOF

xray-node installed.

Panel URL and API token were auto-configured from 3x-ui when possible.
Credentials: /etc/x-ui/install-result.env

1) Verify panel access:
   xray-node inbound list

2) Examples:
   xray-node inbound list
   xray-node inbound apply ${INSTALL_DIR}/configs/inbounds/vless-reality.yaml
   xray-node client add --inbound vless-reality --email user@node
   xray-node client stats --inbound vless-reality --email user@node

3) HTTP API (local):
   curl -H "X-API-Key: <key from config>" http://127.0.0.1:9472/healthz

5) Uninstall everything:
   sudo xray-node uninstall
   # or: curl -fsSL .../scripts/uninstall.sh | sudo bash -s --

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
