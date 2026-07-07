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
# 3x-ui SSL during install: ip | domain | none (default: ip)
XUI_SSL_MODE="${XRAY_NODE_XUI_SSL_MODE:-ip}"
XUI_ACME_HTTP_PORT="${XUI_ACME_HTTP_PORT:-80}"
XUI_FOLDER="${XUI_FOLDER:-/usr/local/x-ui}"
XUI_INSTALL_RESULT="/etc/x-ui/install-result.env"

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
  export XUI_SSL_MODE
  export XUI_ACME_HTTP_PORT
  if [[ -n "${XUI_SSL_IPV6:-}" ]]; then
    export XUI_SSL_IPV6
  fi
  if [[ -n "${XUI_ACME_EMAIL:-}" ]]; then
    export XUI_ACME_EMAIL
  fi
  if [[ -n "${XUI_SERVER_IP:-}" ]]; then
    export XUI_SERVER_IP
  fi

  if command -v x-ui >/dev/null 2>&1; then
    echo "3x-ui already installed"
    setup_xui_ssl_if_needed
    return
  fi

  echo "Installing 3x-ui (SSL mode: ${XUI_SSL_MODE})..."
  bash <(curl -Ls https://raw.githubusercontent.com/mhsanaei/3x-ui/master/install.sh)
  setup_xui_ssl_if_needed
}

detect_server_ip() {
  if [[ -n "${XUI_SERVER_IP:-}" ]]; then
    echo "${XUI_SERVER_IP}"
    return
  fi
  local ip=""
  ip="$(curl -4 -fsS --max-time 5 https://api.ipify.org 2>/dev/null || true)"
  if [[ -z "${ip}" ]]; then
    ip="$(curl -4 -fsS --max-time 5 https://ifconfig.me 2>/dev/null || true)"
  fi
  if [[ -z "${ip}" ]]; then
    ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
  fi
  echo "${ip}"
}

xui_has_ssl() {
  if [[ -f /root/cert/ip/fullchain.pem && -f /root/cert/ip/privkey.pem ]]; then
    return 0
  fi
  if x-ui setting -getCert true 2>/dev/null | grep -Eq 'cert: .+'; then
    return 0
  fi
  if x-ui settings 2>/dev/null | grep -qiE 'SSL certificate configured'; then
    return 0
  fi
  if [[ -f "${XUI_INSTALL_RESULT}" ]]; then
    # shellcheck disable=SC1090
    source "${XUI_INSTALL_RESULT}"
    [[ "${XUI_ACCESS_URL:-}" == https://* ]] && return 0
  fi
  return 1
}

install_acme_sh() {
  if [[ -x "${HOME}/.acme.sh/acme.sh" ]]; then
    return 0
  fi
  echo "Installing acme.sh..."
  curl -fsS https://get.acme.sh | sh >/dev/null
}

setup_xui_ssl_ip() {
  local ipv4="${1:-}"
  local ipv6="${2:-${XUI_SSL_IPV6:-}}"
  local web_port="${XUI_ACME_HTTP_PORT:-80}"
  local cert_dir="/root/cert/ip"
  local xui_bin="${XUI_FOLDER}/x-ui"

  if [[ -z "${ipv4}" ]]; then
    ipv4="$(detect_server_ip)"
  fi
  if [[ -z "${ipv4}" ]]; then
    echo "Could not detect server IPv4 for Let's Encrypt IP certificate" >&2
    return 1
  fi

  echo "Setting up Let's Encrypt IP certificate for ${ipv4}..."
  echo "Port ${web_port} must be reachable from the internet (HTTP-01)."

  install_acme_sh
  if [[ ! -x "${HOME}/.acme.sh/acme.sh" ]]; then
    echo "acme.sh is not available" >&2
    return 1
  fi

  systemctl stop x-ui 2>/dev/null || true
  mkdir -p "${cert_dir}"

  local domain_args="-d ${ipv4}"
  if [[ -n "${ipv6}" ]]; then
    domain_args="${domain_args} -d ${ipv6}"
  fi

  local reload_cmd="systemctl restart x-ui 2>/dev/null || true"
  "${HOME}/.acme.sh/acme.sh" --set-default-ca --server letsencrypt --force >/dev/null 2>&1
  if [[ -n "${XUI_ACME_EMAIL:-}" ]]; then
    "${HOME}/.acme.sh/acme.sh" --register-account -m "${XUI_ACME_EMAIL}" >/dev/null 2>&1 || true
  fi

  if ! "${HOME}/.acme.sh/acme.sh" --issue \
    ${domain_args} \
    --standalone \
    --server letsencrypt \
    --certificate-profile shortlived \
    --days 6 \
    --httpport "${web_port}" \
    --force; then
    echo "Failed to issue IP certificate. Ensure port ${web_port} is open." >&2
    return 1
  fi

  "${HOME}/.acme.sh/acme.sh" --installcert --force -d "${ipv4}" \
    --key-file "${cert_dir}/privkey.pem" \
    --fullchain-file "${cert_dir}/fullchain.pem" \
    --reloadcmd "${reload_cmd}" >/dev/null 2>&1 || true

  if [[ ! -f "${cert_dir}/fullchain.pem" || ! -f "${cert_dir}/privkey.pem" ]]; then
    echo "Certificate files were not created" >&2
    return 1
  fi

  chmod 600 "${cert_dir}/privkey.pem" 2>/dev/null || true
  chmod 644 "${cert_dir}/fullchain.pem" 2>/dev/null || true
  "${HOME}/.acme.sh/acme.sh" --upgrade --auto-upgrade >/dev/null 2>&1 || true

  if [[ -x "${xui_bin}" ]]; then
    "${xui_bin}" cert -webCert "${cert_dir}/fullchain.pem" -webCertKey "${cert_dir}/privkey.pem"
  else
    x-ui cert -webCert "${cert_dir}/fullchain.pem" -webCertKey "${cert_dir}/privkey.pem"
  fi

  systemctl restart x-ui 2>/dev/null || true
  echo "Let's Encrypt IP certificate installed (auto-renews, ~6 days validity)."
}

setup_xui_ssl_if_needed() {
  if [[ "${XUI_SSL_MODE}" == "none" ]]; then
    return 0
  fi
  if [[ "${XUI_SSL_MODE}" != "ip" ]]; then
    echo "Only XUI_SSL_MODE=ip is handled by xray-node install script (got: ${XUI_SSL_MODE})"
    return 0
  fi
  if xui_has_ssl; then
    echo "3x-ui SSL already configured"
    return 0
  fi
  setup_xui_ssl_ip "$(detect_server_ip)" || {
    echo "Warning: IP SSL setup failed; panel remains on HTTP." >&2
    return 0
  }
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
  if xui_has_ssl; then
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
