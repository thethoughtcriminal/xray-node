# xray-node — техническое описание системы

Документ описывает VPN-ноду на базе **3x-ui + Xray** и слой управления **xray-node**. По нему можно воссоздать систему с нуля без доступа к исходному репозиторию.

**Репозиторий:** https://github.com/thethoughtcriminal/xray-node  
**Модуль Go:** `github.com/thethoughtcriminal/xray-node`  
**Версия Go:** 1.22+

---

## 1. Назначение и границы системы

### 1.1 Что делает система

`xray-node` — слой управления **одной VPN-нодой** на VPS: inbounds, клиенты, трафик, CLI + HTTP API.

### 1.2 Принцип работы ноды

`xray-node` — **тонкая обёртка** над REST API панели [3x-ui](https://github.com/MHSanaei/3x-ui) (v3.4+). Сам Xray запускается и конфигурируется 3x-ui; `xray-node` не управляет процессом Xray напрямую.

```
Администратор / внешняя автоматизация
        │  X-API-Key
        ▼
  xray-node HTTP API (:9472, localhost)
        │  Bearer token
        ▼
  3x-ui панель (:PORT/BASE_PATH)
        │  управляет config.json
        ▼
  Xray-core (+ Hysteria2 при необходимости)
        │
        ▼
  Клиенты VPN (VLESS Reality, Hysteria2, …)
```

---

## 2. Архитектура xray-node (код)

### 2.1 Структура проекта

```
xray-node/
├── cmd/xray-node/main.go          # точка входа
├── internal/
│   ├── api/                       # HTTP API (chi router)
│   ├── cli/                       # Cobra CLI
│   ├── config/                    # загрузка config.yaml
│   ├── inbound/                   # YAML-спеки inbounds, overrides, Reality keys
│   ├── panel/                     # клиент 3x-ui REST API
│   └── service/                   # бизнес-логика (Node)
├── configs/
│   ├── config.example.yaml
│   └── inbounds/
│       ├── vless-reality.yaml
│       └── hysteria2.yaml
├── scripts/
│   ├── install.sh                 # полная установка VPS
│   └── uninstall.sh
├── docs/
│   └── TECHNICAL.md               # этот документ
├── go.mod
└── Makefile
```

### 2.2 Слои приложения

```
CLI / HTTP API
      │
      ▼
service.Node          ← оркестрация: ApplyInbound, AddClient, …
      │
      ▼
panel.PanelClient     ← HTTP к 3x-ui (/panel/api/…)
      │
      ▼
3x-ui + Xray
```

**Зависимости Go:**

| Пакет | Назначение |
|-------|------------|
| `github.com/spf13/cobra` | CLI |
| `github.com/go-chi/chi/v5` | HTTP router |
| `github.com/google/uuid` | UUID клиентов |
| `gopkg.in/yaml.v3` | парсинг inbound YAML |

### 2.3 Ключевые типы

**`inbound.Spec`** — декларативное описание inbound (YAML → JSON для панели):

- `remark` — уникальный идентификатор (ключ upsert)
- `protocol`, `port`, `listen`, `enable`, `tag`
- `settings`, `streamSettings`, `sniffing`, `allocate` — произвольные `map[string]any`

**`panel.Inbound`** — ответ 3x-ui; поля `settings` / `streamSettings` могут приходить как **строка JSON** или **вложенный объект** (3x-ui v3.4) — обрабатывается типом `panel.JSONField`.

**`service.Node`** — единая точка бизнес-логики для CLI и API.

---

## 3. Интеграция с 3x-ui

### 3.1 Версия и особенности API

Целевая версия: **3x-ui v3.4.x**.

| Аспект | Детали |
|--------|--------|
| Авторизация | `Authorization: Bearer <API_TOKEN>` |
| Base path | Панель может иметь случайный путь: `https://IP:PORT/BASE_PATH/` |
| Токен | Создаётся при установке; plaintext только один раз → `/etc/x-ui/install-result.env` |
| Клиенты (v3.4+) | Новый API: `/panel/api/clients/*` (не `/panel/api/inbounds/addClient`) |

### 3.2 Используемые эндпоинты 3x-ui

| Операция | Метод | Путь |
|----------|-------|------|
| Список inbounds | GET | `/panel/api/inbounds/list` |
| Создать inbound | POST | `/panel/api/inbounds/add` |
| Обновить inbound | POST | `/panel/api/inbounds/update/{id}` |
| Добавить клиента | POST | `/panel/api/clients/add` |
| Обновить клиента | POST | `/panel/api/clients/update/{email}` |
| Трафик клиента | GET | `/panel/api/clients/traffic/{email}` |

**Тело `POST /panel/api/clients/add`:**

```json
{
  "client": {
    "email": "user@xray-node",
    "enable": true,
    "flow": "xtls-rprx-vision",
    "uuid": "опционально"
  },
  "inboundIds": [1]
}
```

При отсутствии v3.4 API (`HTTP 404`) — fallback на legacy `/panel/api/inbounds/addClient`.

### 3.3 Конфигурация panel.url

В `/etc/xray-node/config.yaml` URL панели должен включать **полный base path**:

```yaml
panel:
  url: https://127.0.0.1:22847/yF8UENTYO0hfLb63sw
  token: <из install-result.env>
  insecure_tls: true   # для self-signed / LE IP cert
```

При установке `install.sh` читает `/etc/x-ui/install-result.env` и прописывает `url`, `token` автоматически.

---

## 4. Конфигурация xray-node

### 4.1 `/etc/xray-node/config.yaml`

```yaml
panel:
  url: https://127.0.0.1:2053/BASE_PATH   # схема + порт + base path
  token: API_TOKEN_FROM_3XUI
  # username / password — запасной вариант (сессия), обычно не нужны
  insecure_tls: true                       # не проверять TLS сертификат панели

api:
  listen: 127.0.0.1:9472                   # только localhost по умолчанию
  key: RANDOM_HEX_48_CHARS                 # заголовок X-API-Key
```

| Поле | Описание |
|------|----------|
| `panel.url` | Базовый URL панели **без** trailing `/panel/api` |
| `panel.token` | Bearer-токен из Settings → Authentication → API Tokens |
| `panel.insecure_tls` | `true` для `127.0.0.1` с самоподписанным / LE IP сертификатом |
| `api.listen` | Адрес HTTP API xray-node |
| `api.key` | Секрет для автоматизации; генерируется при `install.sh` |

### 4.2 Файлы на VPS после установки

| Путь | Назначение |
|------|------------|
| `/usr/local/bin/xray-node` | бинарник |
| `/opt/xray-node/` | исходники / шаблоны |
| `/etc/xray-node/config.yaml` | конфиг (mode 600) |
| `/etc/systemd/system/xray-node.service` | systemd unit |
| `/usr/local/x-ui/` | 3x-ui |
| `/etc/x-ui/x-ui.db` | SQLite БД панели |
| `/etc/x-ui/install-result.env` | креды установки (mode 600) |
| `/root/cert/ip/` | LE IP сертификат панели (если включён SSL) |
| `~/.acme.sh/` | acme.sh для автообновления |

---

## 5. Inbound-шаблоны

### 5.1 VLESS Reality (`configs/inbounds/vless-reality.yaml`)

Целевой профиль (production):

| Параметр | Значение |
|----------|----------|
| remark | `vless-reality` |
| port | `8443` |
| tag | `in-8443-tcp` |
| target | `deepl.com:443` |
| serverNames | `www.deepl.com` |
| fingerprint | `qq` |
| flow (клиент) | `xtls-rprx-vision` (авто при add) |
| sniffing | `enabled: false` |

**Reality keys:** при `inbound apply`, если `privateKey` / `publicKey` пустые:

1. При обновлении — сохраняются существующие ключи из панели
2. При создании — генерируются через `xray x25519` (`/usr/local/x-ui/bin/xray-linux-amd64`)

**Интерактивный apply** (в TTY):

```bash
xray-node inbound apply configs/inbounds/vless-reality.yaml
# Port [8443]:
# SNI [www.deepl.com]:
```

Флаги: `--port`, `--sni`, `--non-interactive`.

> При интерактивном SNI и `target`, и `serverNames` выставляются в одно значение. Для точного шаблона (разные `target` и `serverNames`) используйте `--non-interactive`.

### 5.2 Hysteria2 (`configs/inbounds/hysteria2.yaml`)

| Параметр | Значение |
|----------|----------|
| remark | `hysteria2` |
| protocol | `hysteria` |
| port | `10443` (UDP) |
| tls serverName | `example.com` — **заменить** на реальный домен |

**Вручную после установки:** в панели 3x-ui → inbound hysteria2 → **Set Cert from Panel** (TLS обязателен).

### 5.3 Семантика `inbound apply`

1. Загрузить YAML → `inbound.Spec`
2. Применить overrides (port, SNI) при наличии
3. `EnsureRealityKeys()` для VLESS Reality
4. Найти inbound по `remark`
5. **Создать** или **обновить** через API панели
6. При обновлении: сохранить существующих `clients`, если `settings.clients` пуст в YAML

---

## 6. Управление клиентами

### 6.1 Идентификация клиента

В Xray/3x-ui клиент идентифицируется полем **`email`** (произвольная строка, не обязательно e-mail).

Для учёта трафика на нескольких нодах **один и тот же `email`** должен использоваться на всех нодах (соглашение для внешних систем учёта).

### 6.2 Автоматические поля при `client add`

| Протокол | Поле | Поведение |
|----------|------|-----------|
| vless | `id` (UUID) | генерируется, если не указан `--uuid` |
| vless | `flow` | `xtls-rprx-vision` по умолчанию |
| vless | `subId` | 16 символов UUID |
| hysteria | `auth` | UUID или `--auth` |

### 6.3 CLI

```bash
# Добавить клиента
xray-node client add --inbound vless-reality --email user@xray-node

# Тот же UUID на Hy2 (опционально)
xray-node client add --inbound hysteria2 --email user@xray-node --uuid <UUID>

xray-node client enable  --inbound vless-reality --email user@xray-node
xray-node client disable --inbound vless-reality --email user@xray-node
xray-node client stats   --inbound vless-reality --email user@xray-node
```

### 6.4 Учёт трафика

- Xray считает трафик по **`email`** на ноде
- `client stats` → `GET /panel/api/clients/traffic/{email}`

---

## 7. HTTP API xray-node

**Базовый URL:** `http://127.0.0.1:9472`  
**Авторизация:** заголовок `X-API-Key: <api.key>` (кроме `/healthz`)

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/healthz` | `{"status":"ok"}` |
| GET | `/inbounds` | список inbounds |
| POST | `/inbounds/apply` | JSON body = `inbound.Spec` (финальные port/SNI; без интерактивных override как в CLI) |
| POST | `/clients` | добавить клиента |
| POST | `/clients/{email}/enable?inbound=remark` | включить |
| POST | `/clients/{email}/disable?inbound=remark` | выключить |
| GET | `/clients/{email}/stats?inbound=remark` | трафик |

**Пример добавления клиента:**

```bash
curl -s -X POST \
  -H "X-API-Key: $KEY" \
  -H "Content-Type: application/json" \
  http://127.0.0.1:9472/clients \
  -d '{"inbound_remark":"vless-reality","email":"user@xray-node"}'
```

**Коды ошибок** (тело `{"error":"..."}`):

| HTTP | Когда |
|------|-------|
| 400 | невалидный JSON, spec, отсутствует `inbound` query |
| 401 | неверный `X-API-Key` |
| 404 | inbound или client не найден |
| 409 | клиент уже существует |
| 502 | ошибка 3x-ui или прочий сбой upstream |

---

## 8. Установка с нуля (VPS)

### 8.1 Требования

- Ubuntu/Debian (или RHEL с `dnf`)
- root
- Открытые порты:
  - **80/TCP** — LE IP сертификат панели (если `XRAY_NODE_XUI_SSL_MODE=ip`)
  - **8443/TCP** — VLESS Reality
  - **10443/UDP** — Hysteria2
  - Порт панели 3x-ui (случайный при установке)

### 8.2 Одной командой

```bash
curl -fsSL https://raw.githubusercontent.com/thethoughtcriminal/xray-node/main/scripts/install.sh | sudo bash
```

### 8.3 Что делает `install.sh`

```
1. install_deps          → curl, git, golang-go
2. install_3xui          → 3x-ui + SSL (по умолчанию LE IP)
3. clone_or_update_repo  → /opt/xray-node
4. build_binary          → /usr/local/bin/xray-node
5. write_config          → /etc/xray-node/config.yaml + panel autoconfig
6. write_systemd         → xray-node.service
7. apply_default_inbounds → vless-reality + hysteria2 (--non-interactive)
```

### 8.4 Переменные окружения установки

| Переменная | Default | Описание |
|------------|---------|----------|
| `XRAY_NODE_REPO` | GitHub URL | репозиторий |
| `XRAY_NODE_INSTALL_DIR` | `/opt/xray-node` | каталог |
| `XRAY_NODE_APPLY_INBOUNDS` | `1` | применить шаблоны |
| `XRAY_NODE_XUI_SSL_MODE` | `ip` | SSL панели: `ip` / `none` (режим `domain` — только при первой установке 3x-ui, см. их install.sh) |
| `XUI_ACME_HTTP_PORT` | `80` | порт HTTP-01 |
| `XUI_ACME_EMAIL` | — | email для Let's Encrypt |
| `XUI_SERVER_IP` | auto | публичный IPv4 |
| `XUI_SSL_IPV6` | — | добавить IPv6 в сертификат |

```bash
# без SSL на панели
sudo XRAY_NODE_XUI_SSL_MODE=none ./scripts/install.sh
```

### 8.5 SSL панели (Let's Encrypt IP)

Режим `domain` в `install.sh` xray-node **не реализован** — для доменного сертификата используйте установку 3x-ui напрямую или настройте SSL в панели вручную.

- Профиль **shortlived** (~6 дней), автообновление через `acme.sh`
- Сертификаты: `/root/cert/ip/fullchain.pem`, `privkey.pem`
- При повторном запуске `install.sh` на уже установленной панели без SSL — попытка выпустить сертификат

### 8.6 Удаление

```bash
sudo xray-node uninstall
# или
curl -fsSL .../scripts/uninstall.sh | sudo bash -s -- --yes
```

Удаляет: xray-node, `/opt/xray-node`, `/etc/xray-node`, 3x-ui, Xray.  
Флаг `--keep-3xui` — оставить панель.

### 8.7 Креды после установки

```bash
cat /etc/x-ui/install-result.env
x-ui settings
```

Содержит: `XUI_USERNAME`, `XUI_PASSWORD`, `XUI_PANEL_PORT`, `XUI_WEB_BASE_PATH`, `XUI_ACCESS_URL`, `XUI_API_TOKEN`.

---

## 9. Сборка и разработка

```bash
git clone https://github.com/thethoughtcriminal/xray-node.git
cd xray-node
go mod download
make build                    # → bin/xray-node
go test ./...

# CI: .github/workflows/ci.yml (go test + go build на push/PR в main)

# локальный запуск API
./bin/xray-node serve --config configs/config.example.yaml
```

**systemd unit** (production):

```ini
[Unit]
Description=xray-node management API
After=network.target x-ui.service
Wants=x-ui.service

[Service]
Type=simple
ExecStart=/usr/local/bin/xray-node serve --config /etc/xray-node/config.yaml
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
```

---

## 10. Безопасность

| Компонент | Рекомендация |
|-----------|--------------|
| xray-node API | Только `127.0.0.1`; не открывать наружу без TLS + firewall |
| 3x-ui панель | SSL (LE IP или домен); сложный base path; сменить дефолтный пароль |
| `config.yaml` | mode `600`, содержит секреты |
| `install-result.env` | mode `600`, хранить локально |
| API token 3x-ui | Не коммитить; при утере — создать новый в панели |

**Доступ к xray-node API с другого хоста:**

```bash
# вариант 1: SSH tunnel
ssh -L 9472:127.0.0.1:9472 root@NODE_IP

# вариант 2: listen 0.0.0.0 + firewall только с доверенных IP
```

---

## 11. Диагностика

| Симптом | Причина | Решение |
|---------|---------|---------|
| `connection refused :2053` | неверный порт / панель не запущена | `x-ui settings`, `systemctl status x-ui` |
| `panel GET … HTTP 404` | неверный token или base path | проверить `panel.url`, пересоздать token |
| `cannot unmarshal settings` | старая версия xray-node | обновить бинарник (JSONField fix) |
| `addClient HTTP 404` | 3x-ui < 3.4 без fallback | обновить 3x-ui |
| зависание после Port | stdin не TTY | `--port --sni` или SSH `-t` |
| socket hang up :9472 снаружи | API на localhost | SSH tunnel или изменить `api.listen` |

**Полезные команды:**

```bash
systemctl status x-ui xray-node
journalctl -u xray-node -n 50
xray-node inbound list
curl -sk -H "Authorization: Bearer $TOKEN" "$PANEL_URL/panel/api/inbounds/list"
curl http://127.0.0.1:9472/healthz
```

---

## 12. Чеклист воссоздания ноды

- [ ] VPS с Ubuntu, root доступ
- [ ] Порт 80 открыт (для SSL панели)
- [ ] `curl -fsSL .../install.sh | bash`
- [ ] Сохранить `/etc/x-ui/install-result.env`
- [ ] `xray-node inbound list` — 2 inbound
- [ ] `xray-node client add --inbound vless-reality --email test@node`
- [ ] Проверить подключение клиентом (Happ / v2rayN)
- [ ] Настроить TLS для hysteria2 в панели
- [ ] Открыть 8443/TCP, 10443/UDP в firewall провайдера

---

## 13. История ключевых решений

| Решение | Обоснование |
|---------|-------------|
| 3x-ui вместо raw Xray | UI, API, мультипротокол, готовый Xray |
| `email` как ID клиента | стандарт 3x-ui / Xray для трафика |
| localhost-only API | безопасность; внешний доступ через SSH tunnel или firewall |
| YAML inbounds | декларативность, git-friendly |
| upsert по `remark` | идемпотентный deploy |
| 3x-ui v3.4 clients API | `addClient` на inbound удалён |
| LE IP cert по умолчанию | HTTPS панели без домена |
| `target` ≠ `serverNames` в шаблоне | маскировка Reality (deepl.com / www.deepl.com) |

---

*Документ актуален для коммита `main` репозитория `thethoughtcriminal/xray-node`. При изменениях сверяйте с `README.md` и исходным кодом.*
