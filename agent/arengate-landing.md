# Arengate Landing Page — Task Description

## Референсы

- **Дизайн:** https://www.tensorstax.com/ — минимализм, без меню, без логотипов
- **Фон:** https://4kwallpapers.com/space/gargantua-black-9621.html — чёрная дыра Гаргантюа

---

## Стек

```
React + TypeScript
Vite               — сборщик
CSS Modules        — стили (или styled-components)
nginx              — деплой (статика)
```

---

## Структура проекта

```
arengate-landing/
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts
├── nginx.conf             — конфиг для деплоя
├── public/
│   ├── bg.jpg             — фон чёрная дыра
│   └── bg.mp4             — опционально видео версия
└── src/
    ├── main.tsx
    ├── App.tsx
    ├── App.module.css
    ├── components/
    │   ├── Hero/
    │   │   ├── Hero.tsx
    │   │   └── Hero.module.css
    │   └── Instructions/
    │       ├── Instructions.tsx
    │       ├── Step.tsx
    │       └── Instructions.module.css
    └── hooks/
        └── useIntersectionObserver.ts   — анимации при скролле
```

---

## Структура страницы

### Секция 1 — Hero (первый экран)

```
┌─────────────────────────────────────┐
│                                     │
│                                     │
│                                     │
│        Arendelle Gate Tech          │  ← крупный заголовок
│                                     │
│     Your gateway to free internet   │  ← подзаголовок
│                                     │
│                                     │
│              ↓                      │  ← scroll hint
└─────────────────────────────────────┘
```

- Фон — чёрная дыра Гаргантюа (изображение или видео)
- Текст по центру экрана (flexbox, min-height: 100vh)
- Никаких кнопок, меню, логотипов, навигации
- Лёгкий glow эффект на заголовке
- Плавное появление при загрузке (fade in)

### Секция 2 — Инструкция по боту

```
┌─────────────────────────────────────┐
│                                     │
│  Как начать                         │
│                                     │
│  01  Напиши боту @arengate_bot      │
│      Нажми /start для начала        │
│                                     │
│  02  Запроси доступ                 │
│      Нажми кнопку "Запросить"       │
│                                     │
│  03  Дождись одобрения              │
│      Администратор рассмотрит       │
│      заявку в течение 24 часов      │
│                                     │
│  04  Получи ссылки                  │
│      VPN подписка + Telegram прокси │
│      придут в личку от бота         │
│                                     │
│  05  Подключись                     │
│      Установи Happ (iOS/Android)    │
│      или любой VLESS клиент         │
│                                     │
└─────────────────────────────────────┘
```

- Тёмный фон (#050510)
- Нумерация крупная, акцентная (синий #6B8CFF)
- Каждый шаг появляется при скролле (IntersectionObserver)
- Текст лаконичный, без воды

---

## Компоненты

### Hero.tsx
```tsx
// Полноэкранная секция с фоном и заголовком
// Props: none
// Анимация: fade in + slight translateY на mount
```

### Step.tsx
```tsx
interface StepProps {
  number: string      // "01", "02" ...
  title: string       // "Напиши боту"
  description: string // "Нажми /start..."
}
// Анимация: появляется при входе в viewport
```

### Instructions.tsx
```tsx
// Секция со списком Step компонентов
// Данные шагов — массив объектов, не хардкод в JSX
```

### useIntersectionObserver.ts
```tsx
// Хук для анимации появления элементов при скролле
// Возвращает ref + isVisible boolean
```

---

## Цветовая палитра

```
Фон hero:       #000000
Фон секции 2:   #050510
Текст:          #FFFFFF
Подзаголовок:   #8888AA
Акцент:         #6B8CFF  (холодный синий — отсылка к Frozen)
Нумерация:      #6B8CFF
Glow:           rgba(107, 140, 255, 0.3)
```

---

## Типографика

```
Шрифт:          Space Grotesk (Google Fonts)
Заголовок:      clamp(48px, 8vw, 100px) — адаптивный
Подзаголовок:   clamp(16px, 2vw, 22px)
Нумерация:      clamp(48px, 6vw, 72px)
Шаг заголовок:  20px / 500
Шаг описание:   16px / 400 / #8888AA
```

---

## Анимации

```
Hero заголовок:   fade in + translateY(-20px → 0) / 1s ease
Hero подзаголовок: fade in задержка 300ms
Scroll hint:      bounce анимация стрелки вниз
Шаги:            fade in + translateX(-30px → 0) при входе в viewport
                 каждый шаг с задержкой 150ms * index
```

---

## Что НЕ нужно

```
✗ Меню / навигация
✗ Логотип
✗ Кнопки CTA
✗ Футер
✗ Цены / тарифы
✗ Отзывы / команда
✗ Соцсети
✗ Куки баннер
✗ React Router (одна страница)
✗ State management (Redux и т.д.)
```

---

## Деплой

### Сборка
```bash
npm run build
# → dist/ папка со статикой
```

### nginx.conf
```nginx
server {
    listen 80;
    server_name arengate.tech www.arengate.tech;
    root /var/www/arengate;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    # Кэширование статики
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff2|mp4)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # Gzip
    gzip on;
    gzip_types text/css application/javascript image/svg+xml;
}
```

### Деплой на VPS
```bash
npm run build
scp -r dist/* root@78.17.39.204:/var/www/arengate/
```

### SSL через Hiddify
Hiddify уже управляет SSL на сервере. Добавить домен `arengate.tech` в панель Hiddify → он автоматически получит Let's Encrypt сертификат.

---

## Git ветки (аналогично vpn-bot)

```
main   ← стабильная версия
alfa   ← текущая разработка
beta   ← тестирование перед релизом
```

---

## Порядок разработки

```
[ ] 1. Инициализировать проект: npm create vite@latest
[ ] 2. Настроить TypeScript + CSS Modules
[ ] 3. Подключить шрифт Space Grotesk
[ ] 4. Сделать Hero секцию с фоном
[ ] 5. Добавить анимацию заголовка
[ ] 6. Сделать компонент Step
[ ] 7. Сделать секцию Instructions
[ ] 8. Добавить IntersectionObserver анимации
[ ] 9. Адаптив (mobile)
[ ] 10. Сборка + деплой на VPS
```

---

## Настроение

```
Холодно. Космос. Тишина.
Чёрная дыра притягивает — как Arendelle притягивает.
Минимализм. Никакого шума.
Только ты и ворота.
```
