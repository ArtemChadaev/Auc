# Frontend: Демо-платформа аукциона (Go + React Router 8 + Tailwind + shadcn/ui)

Фронтенд для демонстрации конкурентных торгов в реальном времени, механики Anti-Sniping, заморозки средств (Hold) и управления ставками.

---

## 1. Запуск проекта

```bash
cd frontend
pnpm install # или npm install
npm run dev
```

Приложение будет доступно по адресу: **`http://localhost:5173/`**

### Сборка и проверка типов:
```bash
npm run typecheck   # Typecheck (TypeScript strict, без ошибок и без any)
npm run build       # Продакшн сборка (React Router 8 SPA)
```

---

## 2. Архитектура и переключение на реальный Go API

Сейчас проект работает полностью на **in-memory моках** (`frontend/app/api/mocks/`), не выполняя реальных сетевых запросов.
Сигнатуры всех функций API (`app/api/`) строго соответствуют Go-бэкенду.

### Как переключить на реальный API:
1. В файлах `app/api/auth.ts`, `app/api/lots.ts`, `app/api/bids.ts`, `app/api/items.ts`, `app/api/profile.ts`, `app/api/docs.ts`, `app/api/realtime.ts` измените флаг:
   ```ts
   const USE_MOCKS = false;
   ```
2. Реализуйте настоящие сетевые вызовы в папке **`app/api/real/`**:
   - `app/api/real/auth.ts` — `loginReal`, `registerReal`, `logoutReal`, `getCurrentUserReal`
   - `app/api/real/lots.ts` — `listLotsReal`, `getLotReal`, `createLotReal`
   - `app/api/real/bids.ts` — `listBidsReal`, `placeBidReal`
   - `app/api/real/items.ts` — `listUserItemsReal`, `createItemReal`
   - `app/api/real/profile.ts` — `listTransactionsReal`, `topUpDailyCoinsReal`
   - `app/api/real/docs.ts` — `listDocumentsReal`, `getDocumentReal`, `recordConsentReal`
   Каждая функция уже содержит комментарий `// TODO(real-api): METHOD /path` с указанием нужного эндпоинта.

---

## 3. Пункты, отсутствующие в текущем Go коде (`// MISSING IN GO`)

Некоторые эндпоинты заложены в архитектуру и моки, но пока отсутствуют в Go коде (согласно `docs/api-contract.md`):
1. `POST /api/lot/{id}/bids` — оформление ставки
2. `GET /api/lot/{id}/bids` — история ставок по лоту
3. `POST /api/lot/` — создание лота продавцом
4. `GET /api/user/items` — список предметов пользователя
5. `POST /api/item/` — создание предмета
6. `GET /api/user/transactions` — история операций (ledger)
7. `GET /api/lot/{id}/stream` — SSE поток событий лота
8. `GET /api/stream/public` — SSE поток каталога лотов
9. `GET /api/stream/user` — SSE поток персональных событий пользователя

---

## 4. Особенности реализации интерфейса

- **Светлая тема**: интерфейс зафиксирован в чистой светлой теме.
- **Адаптивность (Mobile-First)**: оптимизирован под два ключевых экрана — компактные смартфоны (375-390px) и компьютеры (Desktop).
- **Валюта Coins (`🪙`)**: виртуальная валюта, тултипы объясняют механику баланса и Hold.
- **Anti-Sniping**: серверное продление времени торгов при ставке в последние 60 секунд.
- **3-этапная регистрация**: учётные данные, согласия с модалками документов, выбор аватара из 5 мок-вариантов.
