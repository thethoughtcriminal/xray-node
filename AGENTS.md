# AGENTS.md — xray-node

Instructions for AI agents working in this repository.

## Scope

This repo is **one VPN node manager** only: declarative inbounds, client lifecycle, traffic stats, CLI + HTTP API over [3x-ui](https://github.com/MHSanaei/3x-ui) v3.4+.

**Out of scope:** subscription servers, user portals, billing, multi-tenant control planes. Those live in other repositories.

**Source of truth:** [docs/TECHNICAL.md](docs/TECHNICAL.md) — architecture, APIs, install, troubleshooting.

## Architecture (do not break)

```
CLI / HTTP API  →  internal/service.Node  →  internal/panel.PanelClient  →  3x-ui API
```

| Layer | Package | Responsibility |
|-------|---------|----------------|
| Entry | `cmd/xray-node` | `main`, delegates to CLI |
| CLI | `internal/cli` | Cobra commands; thin wrappers |
| HTTP | `internal/api` | chi router; mirrors CLI operations |
| Business logic | `internal/service` | `Node`: ApplyInbound, AddClient, stats, … |
| 3x-ui client | `internal/panel` | REST calls, `JSONField` for settings blobs |
| Inbound specs | `internal/inbound` | YAML load, overrides, Reality key generation |
| Config | `internal/config` | `/etc/xray-node/config.yaml` |

**Rules:**

- New operations: implement in `service.Node`, expose via CLI **and** HTTP API unless explicitly CLI-only.
- Do not call 3x-ui from CLI/API handlers directly — always through `panel.PanelClient`.
- Do not manage Xray process or `config.json` directly — only via 3x-ui panel API.

## 3x-ui integration

- Target panel version: **3x-ui v3.4.x**
- Auth: `Authorization: Bearer <token>`
- `panel.url` must include **full base path**: `https://127.0.0.1:PORT/BASE_PATH` (no `/panel/api` suffix)
- Client CRUD (v3.4+): `/panel/api/clients/add`, `/panel/api/clients/update/{email}`, `/panel/api/clients/traffic/{email}`
- Legacy fallback to `/panel/api/inbounds/addClient` only on HTTP 404
- Inbound list responses may return `settings` / `streamSettings` as **JSON string or object** — use `panel.JSONField`

## Inbound templates

- Templates: `configs/inbounds/vless-reality.yaml`, `configs/inbounds/hysteria2.yaml`
- Upsert key: `remark`
- On update: preserve existing **clients** and **Reality keys** unless explicitly set in YAML
- Reality keys: generate via `xray x25519` (`internal/inbound/reality.go`) when empty
- Default VLESS profile: port `8443`, `target: deepl.com:443`, `serverNames: [www.deepl.com]`, fingerprint `qq`

Interactive `inbound apply` prompts for port/SNI in TTY; `install.sh` uses `--non-interactive`.

## Coding standards

- **Go 1.22+**, module `github.com/thethoughtcriminal/xray-node`
- Match existing style: minimal abstractions, small focused diffs, no unrelated refactors
- Comments only for non-obvious logic (3x-ui quirks, security)
- English for code, identifiers, commit messages, and this file
- Run before finishing Go changes:
  ```bash
  go test ./...
  go build -o /dev/null ./cmd/xray-node
  ```
- Keep `go.sum` committed; run `go mod tidy` after dependency changes

## Shell scripts

| Script | Purpose |
|--------|---------|
| `scripts/install.sh` | 3x-ui + xray-node + SSL (default `XRAY_NODE_XUI_SSL_MODE=ip`) |
| `scripts/uninstall.sh` | Remove stack; supports `--yes`, `--keep-3xui` |

Install script conventions:

- `set -euo pipefail`
- Read 3x-ui creds from `/etc/x-ui/install-result.env`
- `git reset --hard` on update (not `git pull`) to avoid untracked `go.sum` conflicts
- Piped uninstall/confirm: read from `/dev/tty` when stdin is not a TTY

Do not hardcode secrets. Do not commit `/etc/xray-node/config.yaml` or install-result files.

## HTTP API

- Listen default: `127.0.0.1:9472` (localhost only)
- Auth: header `X-API-Key` (except `GET /healthz`)
- Map service errors to HTTP status: validation → 400, not found → 404, conflict → 409, panel/upstream → 502
- When adding endpoints, update `internal/api/server.go`, `README.md`, and `docs/TECHNICAL.md`

## Security

- Never log or print `panel.token`, `api.key`, or contents of `install-result.env`
- Do not expose xray-node API on `0.0.0.0` without explicit user request and firewall guidance
- `insecure_tls: true` is expected for local panel access with LE IP / self-signed certs

## Git and commits

- **Do not commit** unless the user explicitly asks
- **Do not push** unless the user explicitly asks
- Conventional Commits; subject ≤72 chars; explain *why* in body when needed
- No force-push to `main`

## Common pitfalls (already hit in this project)

| Issue | Fix |
|-------|-----|
| `connection refused` on default 2053 | Panel port/base path changed; read `x-ui settings` / `install-result.env` |
| `panel GET … HTTP 404` | Wrong token or missing base path in `panel.url` |
| `cannot unmarshal settings` | Old code expected string-only; use `JSONField` |
| `addClient 404` on 3x-ui 3.4 | Use `/panel/api/clients/add` |
| Prompt hang after port | Use `/dev/tty` for I/O; `--non-interactive` in install |
| `curl \| bash` uninstall silent exit | Confirm via `/dev/tty` or `--yes` |

## VPS testing

If SSH access is available (e.g. host alias `vdsina`), agents may deploy and verify on the user's VPS **only when asked**. Do not store or echo credentials from `/etc/x-ui/install-result.env`.

## Files to update together

When changing behavior, keep docs in sync:

| Change | Update |
|--------|--------|
| CLI command/flags | `README.md`, `docs/TECHNICAL.md` |
| HTTP API | `README.md`, `docs/TECHNICAL.md` |
| Install/uninstall | `scripts/*.sh`, `README.md`, `docs/TECHNICAL.md` |
| Inbound defaults | `configs/inbounds/*.yaml`, `docs/TECHNICAL.md` |
| 3x-ui API paths | `internal/panel/client.go`, `docs/TECHNICAL.md` |

## What not to do

- Add subscription-server features to this repo
- Add heavy frameworks, ORMs, or databases to xray-node
- Replace 3x-ui with direct Xray config editing
- Broad refactors unrelated to the task
- Create empty tests that only assert the obvious
- Add `CHANGELOG.md` or extra docs unless requested
