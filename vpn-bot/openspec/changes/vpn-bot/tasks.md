## 1. User access request (`user-access-request`)

- [ ] 1.1 Пройти сценарии `/start` и callback `req`: тексты, клавиатура, рассылка админам — сверить с `specs/user-access-request/spec.md`.
- [ ] 1.2 Проверить ветки `banned`, `active`+UUID и успешный `pending` в `bot/handlers/callbacks.go` и `store.SetPendingRequest`.

## 2. Admin moderation (`admin-moderation`)

- [ ] 2.1 Ручная проверка `/stats` без аргумента, с `@username`, `/users`, `/approve` — доступ только для `ADMIN_IDS`.
- [ ] 2.2 Проверить callback `a:` / `x:`: права админа, idempotent reject (`Уже обработано`), сообщения пользователю.
- [ ] 2.3 Проверить `/test`: только админ, при active+UUID — полный пакет выдачи как у пользователя; без записи — понятное сообщение.

## 3. Hiddify provisioning (`hiddify-provisioning`)

- [ ] 3.1 Пройти одобрение с тестовым Hiddify: создание пользователя, лимиты из `USER_PACKAGE_DAYS` / `USER_USAGE_LIMIT_GB`, запись UUID и `ActivateUser`.
- [ ] 3.2 Проверить повторное одобрение уже `active` с UUID (без дубликата в Hiddify, переотправка ссылок).
- [ ] 3.3 Проверить отказ для `banned` и недопустимых статусов.

## 4. User subscription status (`user-subscription-status`)

- [ ] 4.1 `/status` без строки в БД, с `pending`/пустым UUID, с `active`+UUID (успех и ошибка Hiddify).

## 5. Subscription revocation (`subscription-revocation`)

- [ ] 5.1 `/revoke` для админа: пользователь с UUID (удаление в Hiddify + `banned` + DM), без UUID (только БД + текст админу).
- [ ] 5.2 Негатив: не-админ, несуществующий username.

## 6. Зафиксировать расхождения и решения

- [ ] 6.1 Если при аудите найдено поведение вне спек — обновить соответствующий `spec.md` или завести новый change под правку кода.
- [ ] 6.2 Закрыть или перенести в backlog открытые вопросы из `design.md` (статус reject vs `banned`, повторная заявка после ban).

## 7. Connection link delivery (`connection-link-delivery`)

- [ ] 7.1 Обновить формирование Telegram Proxy URL: отправлять только домен с префиксом `users.` (без IP в пользовательском сообщении).
- [ ] 7.2 Добавить/проверить выдачу трех VPN-ссылок: WireGuard, Full Xray, All configs.
- [ ] 7.3 Реализовать и проверить порядок отправки: VPN payload -> Telegram Proxy payload -> support-tag сообщение.
- [ ] 7.4 Добавить callback-кнопки скачивания "WireGuard" и "Full Xray" в VPN-сообщение.
- [ ] 7.5 По callback выполнять server-side загрузку конфига по ссылке и отправлять документом `.txt`.
- [ ] 7.6 Реализовать маску имени файла `<user>_<protocol>.txt` и проверить кейсы username/id fallback.

## 8. Rich media cards (`rich-media-cards`)

- [ ] 8.1 Подключить зависимость `golang.org/x/image` и добавить модуль генерации карточек.
- [ ] 8.2 Добавить брендовые фоновые assets (в стиле `arengate-landing`) и рендер заголовков "VPN" / "Telegram Proxy".
- [ ] 8.3 Реализовать fallback на текстовую выдачу ссылок при ошибке генерации изображения.

## 9. User self service (`user-self-service`)

- [ ] 9.1 Добавить пользовательскую команду перевыпуска конфигов (`/revoke` + alias `/revok`) только для своего Telegram ID.
- [ ] 9.2 Добавить пользовательскую команду `/stats` с минимальной статистикой трафика (used + limit/remaining).
- [ ] 9.3 Добавить финальное сообщение с контактным тегом поддержки (конфигурируемое значение, например `SUPPORT_TAG`).
