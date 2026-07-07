# xray-node

Management layer for a VPN node running [3x-ui](https://github.com/MHSanaei/3x-ui): declarative inbounds, client lifecycle, traffic stats, CLI + HTTP API.

**Full technical specification:** [docs/TECHNICAL.md](docs/TECHNICAL.md)

## Features

- One-shot install script (3x-ui + xray-node)
- Apply/update inbound from YAML (`vless-reality`, `hysteria2` templates included)
- Add client, enable/disable client
- Per-client traffic stats via 3x-ui API
- Local HTTP API for automation

## Quick install (VPS)

```bash
curl -fsSL https://raw.githubusercontent.com/thethoughtcriminal/xray-node/main/scripts/install.sh | sudo bash
```

By default the installer requests a **Let's Encrypt IP certificate** for the 3x-ui panel (`XUI_SSL_MODE=ip`). Port **80** must be open on the VPS.

```bash
# skip panel SSL (HTTP only)
sudo XRAY_NODE_XUI_SSL_MODE=none ./scripts/install.sh

# optional: ACME email, custom public IP, IPv6, alternate HTTP-01 port
sudo XRAY_NODE_XUI_SSL_MODE=ip XUI_ACME_EMAIL=you@example.com ./scripts/install.sh
```

Or from a local clone:

```bash
sudo XRAY_NODE_REPO=file://$(pwd) ./scripts/install.sh
```

## Uninstall

Removes xray-node (service, binary, `/etc/xray-node`, `/opt/xray-node`) and **3x-ui / Xray** by default:

```bash
sudo xray-node uninstall
# or (interactive prompt works when piped):
curl -fsSL https://raw.githubusercontent.com/thethoughtcriminal/xray-node/main/scripts/uninstall.sh | sudo bash -s --
# without prompt:
curl -fsSL https://raw.githubusercontent.com/thethoughtcriminal/xray-node/main/scripts/uninstall.sh | sudo bash -s -- --yes
```

Options:

```bash
sudo xray-node uninstall -y              # no confirmation
sudo xray-node uninstall --keep-3xui     # remove only xray-node
```

After install:

1. Open 3x-ui panel, create **API token** (Settings → API).
2. Put token into `/etc/xray-node/config.yaml` → `panel.token`.
3. `sudo systemctl restart xray-node`

## Config

`/etc/xray-node/config.yaml`:

```yaml
panel:
  url: https://127.0.0.1:2053
  token: YOUR_3XUI_API_TOKEN
  insecure_tls: true

api:
  listen: 127.0.0.1:9472
  key: YOUR_NODE_API_KEY
```

## CLI

```bash
# Inbounds
xray-node inbound apply configs/inbounds/vless-reality.yaml
# prompts: Port [8443], SNI [www.deepl.com]
xray-node inbound apply configs/inbounds/vless-reality.yaml --port 8443 --sni www.deepl.com
xray-node inbound apply configs/inbounds/hysteria2.yaml
xray-node inbound list

# Clients (same email on every node for shared traffic accounting)
xray-node client add --inbound vless-reality --email user@xray-node
xray-node client add --inbound hysteria2 --email user@xray-node --uuid <same-uuid>

xray-node client disable --inbound vless-reality --email user@xray-node
xray-node client enable --inbound vless-reality --email user@xray-node
xray-node client stats --inbound vless-reality --email user@xray-node
```

`inbound apply` updates an existing inbound by `remark` and **keeps existing clients** unless `settings.clients` is set in YAML. In a terminal it prompts for **port** (all inbounds) and **SNI** (VLESS Reality). Reality **x25519 keys** are generated automatically via `xray x25519` when missing. Use `--port`, `--sni`, or `--non-interactive` to skip prompts.

For VLESS Reality you no longer need to generate keys in the 3x-ui UI. For Hysteria2, set TLS cert in panel (`Set Cert from Panel`).

## HTTP API

Auth: header `X-API-Key: <api.key from config>`

| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | Health check (no auth) |
| GET | `/inbounds` | List inbounds |
| POST | `/inbounds/apply` | JSON body = inbound spec |
| POST | `/clients` | Add client |
| POST | `/clients/{email}/enable?inbound=remark` | Enable |
| POST | `/clients/{email}/disable?inbound=remark` | Disable |
| GET | `/clients/{email}/stats?inbound=remark` | Traffic stats |

Example:

```bash
curl -s -H "X-API-Key: $KEY" http://127.0.0.1:9472/inbounds | jq .

curl -s -X POST -H "X-API-Key: $KEY" -H "Content-Type: application/json" \
  http://127.0.0.1:9472/clients \
  -d '{"inbound_remark":"vless-reality","email":"user@xray-node"}' | jq .
```

## Development

```bash
go test ./...
make build
./bin/xray-node --config configs/config.example.yaml inbound list
```

## GitHub

`gh` was not available on the install machine. Create the remote repo and push:

```bash
gh repo create xray-node --public --source=. --remote=origin --push
# or manually on github.com, then:
git remote add origin git@github.com:YOUR_USER/xray-node.git
git add -A && git commit -m "feat: initial VPN node manager for 3x-ui"
git push -u origin main
```

## License

MIT
