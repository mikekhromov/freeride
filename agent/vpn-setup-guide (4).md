# Arengate VPN System — Setup Guide v6.0

## Стек технологий
- **VPS:** AdminVPS, Финляндия, Ubuntu 22.04
- **Тариф:** Start (4 ядра, 8 GB RAM, 60 GB NVMe, 10 Гбит/с)
- **VPN:** Hiddify Manager v12+ (установлен напрямую на VPS)
- **Прокси Telegram:** MTProxy (встроен в Hiddify)
- **Бот:** Go (systemd сервис, без Docker)
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
            │   Hiddify Manager v12       │
            │   ├── VLESS + Reality       │
            │   ├── Hysteria2             │
            │   ├── MTProxy (встроен)     │
            │   └── REST API v2           │
            │                             │
            │   vpn-bot (Go, systemd)     │
            │   ├── Telegram Bot          │
            │   ├── Hiddify API клиент    │
            │   └── SQLite DB             │
            └─────────────────────────────┘
```

**Почему нет Docker:** Hiddify установлен напрямую на VPS и содержит всё включая MTProxy. Go бот — простой бинарник под systemd. Docker здесь только усложнил бы деплой.

**Почему нет мониторинга лимитов:** MTProxy секрет генерируется из UUID пользователя — удалил пользователя в Hiddify и оба доступа (VPN + MTProxy) автоматически перестают работать. Каждый пользователь сам отвечает за своё использование.

---

## Что где хранится

```
GitHub (публично)           VPS ~/vpn-bot/      Локально (у тебя)
────────────────────        ──────────────────   ─────────────────
Весь исходный код           .env.local           .env.local
vpn-bot.service             data/                (копия .env.local)
.env.example                  └── database.db
.gitignore
README.md
```

---

## Правила Git веток

```
main   ← стабильная версия, только проверенный код
alfa   ← текущие наработки (может быть нестабильным)
beta   ← новые фичи готовые к тестированию
```

### Флоу работы
```
1. Пишешь новую фичу → коммитишь в alfa
2. Фича готова к тесту → создаёшь ветку beta от alfa
3. Тестируешь в beta
4. Всё ок → merge beta → main
5. Деплой на VPS всегда с main
```

---

## Структура репозитория

```
vpn-bot/
├── main.go
├── vpn-bot.service             — systemd unit файл
├── .env.example                — шаблон переменных
├── .gitignore
├── go.mod
├── go.sum
├── README.md
├── config/
│   └── config.go               — конфиг из env переменных
├── db/
│   ├── db.go                   — инициализация SQLite
│   ├── migrations/
│   │   └── 001_init.sql        — SQL схема
│   └── models/
│       └── user.go
├── bot/
│   ├── bot.go                  — инициализация бота
│   ├── handlers/
│   │   ├── start.go
│   │   ├── request.go
│   │   ├── approve.go
│   │   ├── revoke.go
│   │   ├── status.go
│   │   └── users.go
│   └── keyboards/
│       └── keyboards.go        — inline кнопки
└── services/
    └── hiddify/
        └── client.go           — Hiddify REST API клиент
```

---

## .gitignore

```gitignore
# Секреты — никогда в репу
.env.local
*.env

# Данные
data/

# База данных
*.db
*.sqlite

# Go бинарник
vpn-bot

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
HIDDIFY_DOMAIN=https://arengate.tech
HIDDIFY_ADMIN_PATH=ВАШ_ADMIN_PATH
HIDDIFY_CLIENT_PATH=ВАШ_CLIENT_PATH
HIDDIFY_API_KEY=ВАШ_ADMIN_UUID

# Настройки пользователей
USER_PACKAGE_DAYS=90
USER_USAGE_LIMIT_GB=1000

# БД
DB_PATH=./data/database.db
```

---

## Hiddify API (проверено)

### Пути

```
proxy_path_client → User API  (для получения ссылок)
proxy_path_admin  → Admin API (для управления пользователями)
```

### Эндпоинты

```bash
# Список пользователей
GET  /ADMIN_PATH/api/v2/admin/user/
Header: Hiddify-API-Key: ADMIN_UUID

# Создать пользователя
POST /ADMIN_PATH/api/v2/admin/user/
Body: {"name": "ivan", "package_days": 90, "usage_limit_GB": 1000}

# Деактивировать пользователя
PATCH /ADMIN_PATH/api/v2/admin/user/{uuid}/
Body: {"enable": false}

# Удалить пользователя
DELETE /ADMIN_PATH/api/v2/admin/user/{uuid}/

# Профиль пользователя → ссылка подписки VPN
GET /CLIENT_PATH/api/v2/user/me/
Header: Hiddify-API-Key: USER_UUID
→ поле profile_url

# MTProxy ссылки пользователя
GET /CLIENT_PATH/api/v2/user/mtproxies/
Header: Hiddify-API-Key: USER_UUID
→ массив {link, title}
```

### Как работает MTProxy секрет

```
Секрет = "ee" + UUID_без_дефисов + fake_domain_hex

Удалил пользователя → секрет автоматически перестаёт работать
Не нужно управлять секретами отдельно!
```

---

## База данных (SQLite)

Минимальная — только для связи Telegram ↔ Hiddify:

### 001_init.sql

```sql
CREATE TABLE users (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    telegram_id       INTEGER UNIQUE NOT NULL,
    telegram_username TEXT,
    hiddify_uuid      TEXT,           -- UUID в Hiddify (null пока pending)
    status            TEXT DEFAULT 'pending',
                      -- pending / active / banned
    created_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
    approved_by       INTEGER,        -- telegram_id админа
    approved_at       DATETIME
);
```

Всё остальное (трафик, подключения, ссылки) хранится в Hiddify.

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
  1. POST /api/v2/admin/user/ → создаёт пользователя в Hiddify
  2. GET /api/v2/user/me/ → берёт profile_url (ссылка подписки)
  3. GET /api/v2/user/mtproxies/ → берёт MTProxy ссылку
  4. Сохраняет hiddify_uuid в БД (status = active)
  5. Отправляет пользователю:

  ✅ Доступ одобрен!

  🔐 VPN (все устройства):
  [profile_url]
  Установи Happ и добавь подписку по ссылке

  📱 Прокси Telegram:
  [tg://proxy?...]
  Нажми чтобы подключить в Telegram

  ⚠️ Ссылки только для личного использования.
```

### 3. Отзыв доступа
```
Админ → /revoke @ivan
Backend →
  1. DELETE /api/v2/admin/user/{uuid}/ → удаляет из Hiddify
  2. БД: status = banned
  3. VPN и MTProxy автоматически перестают работать

  Пользователю →
    🔴 Ваш доступ отозван.
    Обратитесь к администратору для восстановления.

  Админу →
    ✅ Доступ @ivan отозван.
```

---

## Команды бота

### Пользователь
```
/start        — начало работы, кнопка запроса доступа
/status       — мой статус и ссылки
```

### Админ
```
/users              — список всех пользователей
/approve @ivan      — одобрить доступ
/revoke @ivan       — отозвать доступ
/stats              — статистика (из Hiddify API)
```

---

## systemd сервис (vpn-bot.service)

```ini
[Unit]
Description=Arengate VPN Bot
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=vpnbot
WorkingDirectory=/home/vpnbot
EnvironmentFile=/home/vpnbot/.env.local
ExecStart=/usr/local/bin/vpn-bot
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

---

## Установка бота на VPS (первый раз)

```bash
# 1. Создать пользователя
useradd -m -s /bin/bash vpnbot

# 2. Клонировать репо с ветки main
git clone https://github.com/ВАШ_РЕПО/vpn-bot.git /home/vpnbot/vpn-bot
cd /home/vpnbot/vpn-bot

# 3. Создать .env.local
cp .env.example /home/vpnbot/.env.local
nano /home/vpnbot/.env.local   # заполнить реальные значения

# 4. Создать папку для данных
mkdir -p /home/vpnbot/data

# 5. Собрать бинарник
go build -o vpn-bot .
cp vpn-bot /usr/local/bin/vpn-bot

# 6. Установить systemd сервис
cp vpn-bot.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable vpn-bot
systemctl start vpn-bot

# 7. Проверить
systemctl status vpn-bot
journalctl -u vpn-bot -f
```

---

## Обновление (после изменений в репе)

```bash
cd /home/vpnbot/vpn-bot
git pull origin main
go build -o vpn-bot .
sudo cp vpn-bot /usr/local/bin/vpn-bot
sudo systemctl restart vpn-bot
```

Данные в сохранности — `.env.local` и `data/database.db` не трогаются при обновлении.

---

## Откат к предыдущей версии

```bash
cd /home/vpnbot/vpn-bot
git log --oneline           # смотришь хэши коммитов
git checkout ХЭШ_КОММИТА
go build -o vpn-bot .
sudo cp vpn-bot /usr/local/bin/vpn-bot
sudo systemctl restart vpn-bot
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

### Бот на VPS
```bash
systemctl status vpn-bot          # статус
systemctl restart vpn-bot         # перезапустить
journalctl -u vpn-bot -f          # логи в реальном времени
journalctl -u vpn-bot --since "1 hour ago"  # логи за час
```

### Hiddify на VPS
```bash
systemctl status hiddify-panel    # статус панели
systemctl restart hiddify-panel   # перезапустить
tail -f /opt/hiddify-manager/log/system/hiddify_panel.out.log
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
[ ] 1. Создать GitHub репо с ветками main / alfa / beta
[ ] 2. Создать бота через @BotFather, получить BOT_TOKEN
[ ] 3. Зарегистрировать домен arengate.tech
[ ] 4. Go бот — разработка в ветке alfa:
    [ ] a. config + инициализация БД + миграции
    [ ] b. /start, /request — базовый флоу пользователя
    [ ] c. Hiddify API клиент (create / delete / get links)
    [ ] d. approve / revoke флоу + уведомления
    [ ] e. /status, /users команды
    [ ] f. systemd сервис, деплой на VPS
[ ] 5. Тест → merge alfa → beta → main
[ ] 6. Keenetic: Entware + sing-box (если ещё не готово)
[ ] 7. Landing page: arengate.tech (React + TypeScript)
[ ] 8. End-to-end тест всей системы
```
