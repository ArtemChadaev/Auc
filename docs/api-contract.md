# API Contract (Auction)

Основан на Go-коде репозитория (`internal/`, `cmd/main.go`, `docs/swagger.yaml`, `migrations/init.sql`).
Фронтенд работает полностью на моках, но структуры данных, типы и сигнатуры строго синхронизированы с Go-кодом.

---

## 1. Базовые правила взаимодействия

### Формат ответа сервера (`httpx.Response`)
Все HTTP-ответы Go оборачиваются в единую структуру:
```json
{
  "data": { ... },
  "error": "error message if any"
}
```
- При успехе: HTTP 200/201, поле `data` заполнено, `error` отсутствует или пустой.
- При ошибке: HTTP 4xx/5xx, `error` содержит текст ошибки, `data` отсутствует.
- **Внимание (правило проекта)**: Серверные 5xx ошибки никогда не отдаются пользователю как необработанный сырой сбой; фронтенд преобразует их в безопасные понятные сообщения.

### Формат денег и чисел
- Все денежные суммы (`balance`, `hold`, `start_amount`, `current_bid`, `min_step`, `finish_amount`, `amount`) в Go — **целые числа** (`int` / `int64`).
- Минимальная единица: условные рубли/копейки.
- Фронтенд работает с целыми числами, форматирование в валюту производится строго через утилиты `lib/money.ts`.

### Авторизация
- Header: `Authorization: Bearer <access_token>`
- Refresh token: хранится и передаётся через HttpOnly cookie `refresh_token` (в Go `/api/auth/refresh`).

---

## 2. Сущности / DTO (из Go)

### User
```typescript
interface User {
  id: string; // uuid
  name: string;
  avatar_id?: string | null; // uuid
  email: string;
  balance: number; // int
  hold: number; // int (balance >= hold)
  created_at: string; // ISO 8601 timestamptz
  deleted_at?: string | null;
}
```

### UserSession & Device
```typescript
interface UserDevice {
  browser?: string;
  os?: string;
}

interface UserSession {
  refresh_id: number;
  created_at: string;
  expires_at: string;
  revoked_at?: string | null;
  device?: UserDevice;
}
```

### Item (предмет)
```typescript
type ItemType = 'image' | '3d' | 'audio' | 'video' | 'document' | 'archive';

interface Item {
  id: number; // bigint
  creator_id: string; // uuid
  owner_id: string; // uuid
  s3_id: string; // uuid
  type: ItemType;
  metadata?: Record<string, any>;
  created_at: string;
}
```

### Lot & Lot Description
```typescript
type LotStatus =
  | 'draft'
  | 'pending_review'
  | 'approved'
  | 'active'
  | 'rejected'
  | 'finished'
  | 'cancelled';

type MediaType = 'image' | 'video' | '3d';

interface MediaItem {
  id: string; // uuid
  type: MediaType;
  blur_hash?: string;
  width?: number;
  height?: number;
  duration_sec?: number;
  thumbnail_id?: string | null;
  order: number;
  caption?: string;
}

interface LotDescription {
  text: string;
  short_text?: string;
  media: MediaItem[];
  attributes?: Record<string, string>;
}

interface SellerInfo {
  id: string; // uuid
  name: string;
  avatar_id?: string | null;
  rating?: number;
}

interface WinnerInfo {
  id: string; // uuid
  name: string;
}

interface ItemSummary {
  id: number;
  type: string;
  preview: string; // uuid
  blur_hash?: string;
}

interface Lot {
  id: number; // bigint
  seller_id: string; // uuid
  seller: SellerInfo;
  item_id: number; // bigint
  item_summary?: ItemSummary;
  status: LotStatus;
  category: string;
  title: string;
  description: LotDescription;
  start_amount: number; // int64
  current_bid: number; // int64
  min_step: number; // int64
  finish_amount?: number | null; // int64
  bids_count: number;
  winner?: WinnerInfo | null;
  created_at: string;
  started_at?: string | null;
  ends_at?: string | null;
  finished_at?: string | null;
}
```

### Bid & Hold (из migrations/init.sql)
```typescript
interface Bid {
  id: number;
  lot_id: number;
  user_id: string;
  user_name?: string; // для отображения лидера/истории
  amount: number;
  idempotency_key: string;
  created_at: string;
}

type HoldStatus = 'active' | 'released' | 'captured';

interface Hold {
  id: number;
  lot_id: number;
  user_id: string;
  bid_id: number;
  amount: number;
  status: HoldStatus;
  created_at: string;
}
```

### LedgerEntry (Транзакции)
```typescript
type LedgerType = 'top_up' | 'payout_of_winnings' | 'transfer_to_seller';

interface LedgerEntry {
  id: number;
  user_id: string;
  amount: number; // положительное или отрицательное
  balance_after: number;
  type: LedgerType;
  lot_id?: number | null;
  created_at: string;
}
```

### Documents (Документы)
```typescript
interface Document {
  id: number;
  name: string;
  version: string;
  is_major: boolean;
  text: string;
  published_at: string;
  superseded_at?: string | null;
}

interface UserConsent {
  document_id: number;
  user_id: string;
  created_at: string;
}
```

---

## 3. Таблица эндпоинтов

### Существующие в Go:
| Метод | Путь | Назначение |
|---|---|---|
| `POST` | `/api/auth/register` | Регистрация (name, email, password, uid) |
| `POST` | `/api/auth/login` | Вход (email, password) |
| `POST` | `/api/auth/logout` | Выход (refresh_id) |
| `POST` | `/api/auth/refresh` | Обновление токена |
| `POST` | `/api/auth/session` | Сессии пользователя |
| `GET` | `/api/user/me` | Данные текущего пользователя (баланс, hold) |
| `GET` | `/api/user/me/avatar` | Аватар |
| `PATCH` | `/api/user/name` | Смена имени |
| `GET` | `/api/lot/` | Каталог лотов (фильтры, пагинация) |
| `GET` | `/api/lot/{id}` | Детальная инфа о лоте |
| `GET` | `/api/document/` | Список документов |
| `GET` | `/api/document/{id}` | Документ по ID |
| `GET` | `/api/document/name/{name}` | Документ по имени |
| `POST` | `/api/document/consents` | Согласие с документом |
| `POST` | `/api/upload/url` | Получение S3 presigned URL |
| `POST` | `/api/upload/confirm` | Подтверждение загрузки |
| `POST` | `/api/upload/cancel` | Отмена загрузки |
| `GET` | `/api/upload/status` | Статус загрузки |
| `GET` | `/api/item/{id}/download` | Ссылка на скачивание |
| `POST` | `/api/item/gift` | Передача предмета |

### MISSING IN GO (план/моки):
| Метод | Путь | Назначение |
|---|---|---|
| `POST` | `/api/lot/{id}/bids` | Сделать ставку (`// MISSING IN GO`) |
| `GET` | `/api/lot/{id}/bids` | История ставок лота (`// MISSING IN GO`) |
| `POST` | `/api/lot/` | Создание лота продавцом (`// MISSING IN GO`) |
| `GET` | `/api/user/items` | Список предметов пользователя (`// MISSING IN GO`) |
| `POST` | `/api/item/` | Создание предмета (`// MISSING IN GO`) |
| `GET` | `/api/user/transactions`| История транзакций/hold (`// MISSING IN GO`) |
| `POST` | `/api/user/top-up` | Мок пополнения баланса (`// MISSING IN GO`) |
| `GET` | `/api/time` | Текущее серверное время (`// MISSING IN GO`) |
| `GET` | `/api/lot/{id}/stream` | SSE поток лота (`// MISSING IN GO`) |
| `GET` | `/api/stream/public` | SSE публичный каталог (`// MISSING IN GO`) |
| `GET` | `/api/stream/user` | SSE персональные события пользователя (`// MISSING IN GO`) |
| `POST` | `/api/demo/bots/start` | Демо-боты: запуск (`// MISSING IN GO`) |
| `POST` | `/api/demo/bots/stop` | Демо-боты: остановка (`// MISSING IN GO`) |

---

## 4. Ошибки (Таблица «Код ошибки → Причина → Отображение»)

| Код ошибки | Описание в Go / Причина | Категория (`kind`) | Пользовательское действие / показ |
|---|---|---|---|
| `unauthorized` | Токен отсутствует или невалиден | `auth` | Перенаправить на авторизацию / открыть диалог входа |
| `forbidden` | Нет прав доступа к ресурсу | `auth` | Показать Alert «Действие запрещено» |
| `lot_not_found` | Лот с данным ID не найден | `user` | Страница 404 для лота |
| `lot_not_active` | Аукцион ещё не начался или завершён | `race` | Инлайн предупреждение «Торги по лоту закрыты» |
| `bid_too_low` | Ставка меньше текущей + шаг | `user` | Инлайн под полем: «Минимальная ставка: {min_bid}» |
| `outbid` | Пока оформлялась ставка, вас опередили | `race` | Тост/Alert «Вас опередили!», обновить поле на новую min ставку |
| `insufficient_funds`| Баланс - hold < ставки | `user` | Инлайн/Alert «Недостаточно средств. Пополните баланс» |
| `cannot_bid_own_lot`| Нельзя делать ставку на свой собственный лот | `user` | Предупреждение «Вы не можете ставить на свой лот» |
| `already_highest_bidder`| Текущая лидирующая ставка уже принадлежит вам | `user` | «Ваша ставка уже лидирует» |
| `email_already_exists`| Email занят при регистрации | `user` | Ошибка под полем email |
| `invalid_request_body`| Некорректные параметры запроса | `user` | Подсветка полей валидации |
| `network_error` | Сбой соединения | `system` | Тост «Ошибка соединения, повторная попытка...» |
| `internal_server_error`| Внутренняя ошибка сервера | `system` | Безопасное сообщение «Произошла ошибка, попробуйте позже» |

---

## 5. SSE / Realtime формат событий

События потоков передаются с полями: `id` (sequence/timestamp), `event` (имя события), `data` (JSON-объект):

1. **`useLotStream(lotId)`**:
   - `bid_placed`: `{ lot_id, bid: Bid, current_bid: number, bids_count: number }`
   - `time_extended`: `{ lot_id, ends_at: string, reason: 'anti_sniping' }`
   - `lot_finished`: `{ lot_id, winner: WinnerInfo, finish_amount: number, finished_at: string }`
   - `lot_cancelled`: `{ lot_id }`
2. **`usePublicStream()`**:
   - `lot_updated`: `{ lot_id, current_bid: number, bids_count: number, ends_at?: string }`
   - `lot_started`: `{ lot_id, started_at: string, ends_at: string }`
   - `lot_finished`: `{ lot_id, winner?: WinnerInfo, finish_amount: number }`
3. **`useUserStream()`**:
   - `balance_changed`: `{ balance: number, hold: number, available: number }`
   - `outbid_notification`: `{ lot_id, lot_title: string, your_bid: number, new_bid: number }`
   - `lot_won`: `{ lot_id, lot_title: string, amount: number }`
   - `transaction_created`: `{ transaction: LedgerEntry }`
