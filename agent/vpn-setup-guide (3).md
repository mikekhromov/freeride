# VPN System — Architecture & Setup Guide v4.0

## Стек технологий
- **VPS:** AdminVPS, Финляндия, Ubuntu 22.04
- **Тариф:** Start (4 ядра, 8 GB RAM, 60 GB NVMe, 10 Гбит/с)
- **Контейнеры:** Docker + Docker Compose
- **VPN:** Hiddify (VLESS + Reality, Hysteria2)
- **Прокси Telegram:** MTProxy
- **Бот + Backend:** Go (единый бинарник)
- **База данных:** SQLite
- **Роутер:** Keenetic Hero 4G+ (Entware + sing-box)

---

## Архитектура системы

```
Провайдер → Keenetic Hero 4G+ → все устройства дома
                   ↕ VLESS/Hysteria2
            ┌─────────────────────────────┐
            │   VPS (AdminVPS, Helsinki)  │
            │                             │
            │   Docker Compose            │
            │   ├── hiddify               │
            │   ├── mtproxy               │
            │   └── vpn-bot (Go)          │
            │       ├── Telegram Bot UI   │
            │       ├── Monitor Service   │
            │       └── SQLite DB         │
            └─────────────────────────────┘
```

---

## Что где хранится

```
GitHub (публично)           VPS ~/vpn/          Локально (у тебя)
────────────────────        ───────────────────  ─────────────────
Весь исходный код           .env.local           .env.local
docker-compose.yml          hiddify/             (копия .env.local)
.env.example                mtproxy/data/
.gitignore                  mtproxy/logs/
README.md                   vpn-bot/data/
                              └── database.db
```

---

## Структура репозитория

```
vpn-system/
├── vpn-bot/
│   ├── main.go
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   ├── config/
│   │   └── config.go           — конфиг из env переменных
│   ├── db/
│   │   ├── db.go               — инициализация SQLite
│   │   ├── migrations/
│   │   │   └── 001_init.sql    — SQL схема
│   │   └── models/
│   │       ├── user.go
│   │       ├── secret.go
│   │       ├── warning.go
│   │       └── event.go
│   ├── bot/
│   │   ├── bot.go              — инициализация бота
│   │   ├── handlers/
│   │   │   ├── start.go
│   │   │   ├── request.go
│   │   │   ├── approve.go
│   │   │   ├── revoke.go
│   │   │   ├── reissue.go
│   │   │   ├── stats.go
│   │   │   └── warnings.go
│   │   └── keyboards/
│   │       └── keyboards.go    — inline кнопки
│   └── services/
│       ├── hiddify/
│       │   └── client.go       — Hiddify REST API клиент
│       ├── mtproxy/
│       │   ├── secrets.go      — генерация/отзыв секретов
│       │   └── logs.go         — парсинг логов
│       └── monitor/
│           └── monitor.go      — фоновый мониторинг
├── docker-compose.yml          — только ${ПЕРЕМЕННЫЕ}, без секретов
├── .env.example                — шаблон переменных
├── .gitignore
└── README.md
```

---

## .gitignore

```gitignore
# Секреты — никогда в репу
.env.local
*.env

# Данные сервисов
hiddify/
mtproxy/data/
mtproxy/logs/
vpn-bot/data/

# База данных
*.db
*.sqlite

# Go бинарник
vpn-bot/vpn-bot

# OS
.DS_Store
```

---

## .env.example (в репе — шаблон)

```env
# Telegram Bot
BOT_TOKEN=your_bot_token_here
ADMIN_IDS=your_telegram_id_here

# Hiddify
HIDDIFY_URL=https://YOUR_VPS_IP/YOUR_SECRET/
HIDDIFY_API_KEY=your_hiddify_api_key

# MTProxy
MTPROXY_LOG=/mtproxy/logs/proxy.log
MTPROXY_CONFIG=/mtproxy/data/config

# Настройки
MONITOR_INTERVAL=10m
WARNING_TTL=24h
DEVICE_LIMIT=5
DB_PATH=/app/data/database.db
```

---

## docker-compose.yml (в репе — без секретов)

```yaml
version: "3.8"

services:

  hiddify:
    image: hiddify/hiddify-manager:latest
    container_name: hiddify
    restart: always
    network_mode: host
    volumes:
      - ./hiddify:/opt/hiddify-manager

  mtproxy:
    image: telegrammessenger/proxy:latest
    container_name: mtproxy
    restart: always
    ports:
      - "8443:443"
    volumes:
      - ./mtproxy/data:/data
      - ./mtproxy/logs:/var/log/mtproxy

  vpn-bot:
    build: ./vpn-bot
    container_name: vpn-bot
    restart: always
    volumes:
      - ./vpn-bot/data:/app/data
      - ./mtproxy/logs:/mtproxy/logs:ro
      - ./mtproxy/data:/mtproxy/data
    env_file:
      - .env.local
    depends_on:
      - hiddify
      - mtproxy
```

---

## База данных (SQLite)

### 001_init.sql

```sql
CREATE TABLE users (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    telegram_id       INTEGER UNIQUE NOT NULL,
    telegram_username TEXT,
    status            TEXT DEFAULT 'pending',
                      -- pending / active / banned
    created_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
    approved_by       INTEGER,
    approved_at       DATETIME
);

CREATE TABLE secrets (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id          INTEGER NOT NULL REFERENCES users(id),
    hiddify_user_id  TEXT,
    hiddify_link     TEXT,
    mtproxy_secret   TEXT,
    mtproxy_link     TEXT,
    is_active        BOOLEAN DEFAULT TRUE,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    revoked_at       DATETIME
);

CREATE TABLE secret_events (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    secret_id        INTEGER NOT NULL REFERENCES secrets(id),
    event_type       TEXT NOT NULL,
                     -- connected / disconnected / warning /
                     -- revoked / reissued
    ip_address       TEXT,
    unique_ips_count INTEGER,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE warnings (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    secret_id    INTEGER NOT NULL REFERENCES secrets(id),
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at   DATETIME,             -- +24ч от created_at
    status       TEXT DEFAULT 'pending',
                 -- pending / reissued / revoked / ignored / expired
    admin_action TEXT,
    resolved_at  DATETIME
);
```

---

## Флоу пользователя

### 1. Запрос доступа
```
Пользователь → /start
Бот → "Привет! Нажми кнопку для запроса доступа"
Пользователь → [Запросить доступ]
Бот → сохраняет в БД (status = pending)
Бот → уведомление админу:

  👤 Новый запрос доступа
  Пользователь: @ivan
  ID: 123456789

  [✅ Одобрить] [❌ Отклонить]
```

### 2. Одобрение
```
Админ → [✅ Одобрить]
Backend →
  1. Создаёт пользователя в Hiddify API (device_limit=5)
  2. Генерирует новый MTProxy секрет
  3. Сохраняет в БД (status = active)
  4. Отправляет пользователю:

  ✅ Доступ одобрен!

  🔐 VPN (Hiddify):
  [ссылка подписки]
  Лимит: 5 устройств

  📱 Прокси Telegram:
  [tg://proxy?...]

  ⚠️ Ссылки только для личного использования.
  При превышении лимита секрет будет отозван.
```

### 3. Мониторинг (каждые 10 минут)
```
Monitor горутина →
  Парсит логи MTProxy →
  Считает уникальные IP за 24ч и активные соединения →

  Если уникальных IP > 5 ИЛИ активных > 5:
    → Создаёт warning в БД (expires_at = now + 24ч)

    Админу →
      ⚠️ Превышен лимит!

      Пользователь: @ivan
      Активных соединений: 6
      Уникальных IP за 24ч: 8

      IP адреса:
      • 1.2.3.4 — подключён 2ч назад
      • 5.6.7.8 — подключён 40мин назад
      • 9.10.11.12 — подключён 5мин назад

      [🔄 Перевыпустить] [🔴 Отозвать] [✅ Игнорировать]

    Пользователю →
      ⚠️ Внимание!

      Ваш секрет используется на 6 устройствах.
      Лимит: 5 устройств.

      Если это не вы — сообщите администратору.
      При продолжении секрет будет отозван через 24 часа.

      [🔄 Перевыпустить секрет]
```

### 4. Перевыпуск
```
Админ или пользователь → [🔄 Перевыпустить]
Backend →
  1. Старый MTProxy секрет → удаляется
  2. Старый Hiddify пользователь → деактивируется
  3. Генерируется новый MTProxy секрет
  4. Создаётся новый Hiddify пользователь (device_limit=5)
  5. БД: старый secrets.is_active = false, revoked_at = now
  6. БД: новая запись в secrets
  7. БД: warning.status = reissued
  8. secret_events: event_type = reissued

  Пользователю →
    🔄 Секрет перевыпущен!

    🔐 Новый VPN (Hiddify):
    [новая ссылка]

    📱 Новый прокси Telegram:
    [новая ссылка]

    Старые ссылки больше не работают.

  Админу →
    ✅ Секрет @ivan перевыпущен.
```

### 5. Автоотзыв через 24 часа
```
Monitor горутина →
  Проверяет warnings каждые 10 минут →
  Если status = pending И expires_at < now:
    → Выполняет revoke
    → warning.status = expired

    Пользователю →
      🔴 Ваш секрет отозван.
      Превышение лимита не устранено в течение 24 часов.
      Для восстановления обратитесь к администратору.

    Админу →
      🔴 Секрет @ivan автоматически отозван.
```

---

## Команды бота

### Пользователь
```
/start        — начало работы
/status       — мой статус и ссылки
/reissue      — перевыпустить свой секрет
```

### Админ
```
/stats              — статистика всех пользователей
/stats @ivan        — статистика конкретного пользователя
/approve @ivan      — одобрить доступ
/revoke @ivan       — отозвать доступ
/reissue @ivan      — перевыпустить секрет пользователя
/warnings           — активные предупреждения
/users              — список всех пользователей
```

---

## Установка с нуля (первый раз)

```bash
# 1. Подключиться к VPS
ssh root@ВАШ_IP

# 2. Обновить систему
apt update && apt upgrade -y

# 3. Установить Docker
curl -fsSL https://get.docker.com | sh

# 4. Клонировать репозиторий
git clone https://github.com/ВАШ_РЕПО/vpn-system.git ~/vpn
cd ~/vpn

# 5. Создать .env.local из шаблона
cp .env.example .env.local
nano .env.local     # заполнить реальные значения

# 6. Создать папки для данных (они в .gitignore)
mkdir -p hiddify mtproxy/{data,logs} vpn-bot/data

# 7. Запустить все сервисы
docker compose up -d

# 8. Проверить что всё работает
docker compose ps
docker compose logs -f
```

---

## Обновление (после изменений в репе)

```bash
cd ~/vpn
git pull origin main
docker compose up -d --build vpn-bot
```

Данные в сохранности — `.env.local`, `hiddify/`, `mtproxy/data/`,
`vpn-bot/data/database.db` не трогаются при обновлении.

---

## Откат к предыдущей версии

```bash
cd ~/vpn
git log --oneline           # смотришь хэши коммитов
git checkout ХЭШ_КОММИТА
docker compose up -d --build vpn-bot
```

---

## Часть — Keenetic Hero 4G+ (Entware + sing-box)

### Шаг 1 — Компоненты KeeneticOS
```
http://192.168.1.1 → Система → Компоненты

Установить:
✅ Поддержка открытых пакетов (OPKG)
✅ Интерфейс USB
✅ Файловая система Ext
✅ Протокол IPv6
✅ Модули ядра подсистемы Netfilter
✅ Прокси-сервер DNS-over-HTTPS
✅ Сервер SSH
```

### Шаг 2 — Entware на флешку
```
Подключить USB флешку →
Приложения → Entware → Установить →
Дождаться: [5/5] Установка "Entware" завершена!
```

### Шаг 3 — Подключиться к Entware
```bash
ssh root@192.168.1.1 -p 222
# пароль: keenetic
passwd   # сразу сменить!
```

### Шаг 4 — Установить sing-box
```bash
opkg update && opkg install curl
curl -Lo /opt/sbin/sing-box \
  https://github.com/xray108/sing-box-keenetic/releases/latest/download/sing-box-mipsel
chmod +x /opt/sbin/sing-box
/opt/sbin/sing-box version   # проверить
```

### Шаг 5 — Конфиг sing-box
```bash
mkdir -p /opt/etc/sing-box
nano /opt/etc/sing-box/config.json
```

```json
{
  "log": { "level": "info" },
  "dns": {
    "servers": [
      {
        "tag": "remote",
        "address": "https://1.1.1.1/dns-query",
        "detour": "proxy"
      },
      {
        "tag": "local",
        "address": "https://77.88.8.8/dns-query",
        "detour": "direct"
      }
    ],
    "rules": [{ "outbound": "any", "server": "local" }]
  },
  "inbounds": [{
    "type": "tun",
    "tag": "tun-in",
    "inet4_address": "172.19.0.1/30",
    "auto_route": true,
    "strict_route": false,
    "sniff": true
  }],
  "outbounds": [
    {
      "type": "vless",
      "tag": "proxy",
      "server": "ВАШ_IP_VPS",
      "server_port": 443,
      "uuid": "ВАШ_UUID",
      "flow": "xtls-rprx-vision",
      "tls": {
        "enabled": true,
        "server_name": "ВАШ_SNI",
        "utls": { "enabled": true, "fingerprint": "chrome" },
        "reality": {
          "enabled": true,
          "public_key": "ВАШ_PUBLIC_KEY",
          "short_id": "ВАШ_SHORT_ID"
        }
      }
    },
    { "type": "direct", "tag": "direct" }
  ]
}
```

> Все значения берёшь из ссылки подключения Hiddify.
> Создай отдельного пользователя keenetic в панели Hiddify.

### Шаг 6 — Автозапуск
```bash
nano /opt/etc/init.d/S99singbox
```

```bash
#!/bin/sh
ENABLED=yes
PROCS=sing-box
ARGS="-D /opt/etc/sing-box run"
PREARGS=""
DESC=$PROCS
PATH=/opt/sbin:/opt/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
. /opt/etc/init.d/rc.func
```

```bash
chmod +x /opt/etc/init.d/S99singbox
/opt/etc/init.d/S99singbox start
/opt/etc/init.d/S99singbox status
```

---

## Полезные команды

### Docker на VPS
```bash
docker compose up -d                 # запустить всё
docker compose down                  # остановить всё
docker compose ps                    # статус контейнеров
docker compose logs -f vpn-bot       # логи бота
docker compose logs -f hiddify       # логи Hiddify
docker compose restart vpn-bot       # перезапустить бот
```

### sing-box на роутере
```bash
/opt/etc/init.d/S99singbox start
/opt/etc/init.d/S99singbox stop
/opt/etc/init.d/S99singbox restart
/opt/etc/init.d/S99singbox status
tail -f /opt/var/log/sing-box.log
```

### Проверка VPN
```
Открыть https://2ip.ru — должен показать IP VPS, не провайдера
```

---

## Порядок разработки

```
[ ] 1. Создать GitHub репо, залить базовую структуру проекта
[ ] 2. VPS: Docker, git clone, создать .env.local, mkdir для данных
[ ] 3. Запустить Hiddify + MTProxy, открыть панель Hiddify
[ ] 4. Создать бота через @BotFather, получить BOT_TOKEN
[ ] 5. Go бот — этапы:
    [ ] a. config + инициализация БД + миграции
    [ ] b. /start, /request — базовый флоу пользователя
    [ ] c. Hiddify API клиент (создание/деактивация пользователей)
    [ ] d. MTProxy — генерация секретов, управление конфигом
    [ ] e. approve/revoke флоу + уведомления
    [ ] f. Monitor горутина — парсинг логов каждые 10 минут
    [ ] g. Warning система — варнинги + автоотзыв через 24ч
    [ ] h. Reissue флоу — для админа и пользователя
    [ ] i. /stats, /users, /warnings команды для админа
[ ] 6. Keenetic: Entware + sing-box
[ ] 7. End-to-end тест всей системы
```
