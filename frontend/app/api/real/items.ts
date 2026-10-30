import { api, generateIdempotencyKey } from '~/api/client';
import type { Item, CreateItemInput, DownloadUrlResponse } from '~/api/types';

const STORED_ITEMS_KEY = 'auc_user_items';

/**
 * Очищает устаревший оффлайн-кэш предметов в браузере
 */
export function saveUserItemLocally(_item?: Item): void {
  if (typeof window === 'undefined') return;
  try {
    localStorage.removeItem(STORED_ITEMS_KEY);
  } catch {
    // Ignore
  }
}

/**
 * GET /api/item/{id}/download
 * Возвращает Presigned GET URL для скачивания файла владельцем (не более 1 раза в день)
 */
export async function getItemDownloadUrlReal(id: number): Promise<DownloadUrlResponse> {
  const res = await api.get<{ url: string }>(`/api/item/${id}/download`);
  return {
    url: res.url,
    expires_at: new Date(Date.now() + 3600000).toISOString(),
  };
}

/**
 * POST /api/item/gift
 * Передает предмет новому владельцу с сохранением истории передачи
 */
export async function giftItemReal(
  itemId: number,
  newOwner: string,
  description?: string
): Promise<void> {
  const idempotencyKey = generateIdempotencyKey(`item-gift-${itemId}-${newOwner}`);
  await api.post('/api/item/gift', {
    item_id: Number(itemId),
    new_owner: newOwner,
    description: description || '',
  }, {
    headers: { 'Idempotency-Key': idempotencyKey },
  });
}

/**
 * GET /api/item/me
 * Получение актуального списка предметов текущего авторизованного пользователя из БД
 */
export async function listUserItemsReal(): Promise<Item[]> {
  // Очищаем устаревший локальный кэш, если он остался в браузере
  if (typeof window !== 'undefined') {
    try {
      localStorage.removeItem(STORED_ITEMS_KEY);
    } catch {
      // Ignore
    }
  }

  try {
    const items = await api.get<Item[]>('/api/item/me');
    if (!Array.isArray(items)) return [];
    return items.sort(
      (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    );
  } catch (err) {
    console.warn('Failed to load user items from /api/item/me:', err);
    return [];
  }
}

/**
 * Получение предмета по ID
 */
export async function getItemReal(id: number): Promise<Item> {
  const items = await listUserItemsReal();
  const found = items.find((i) => i.id === id);
  if (found) return found;

  return {
    id,
    creator_id: '',
    owner_id: '',
    s3_id: `s3-${id}`,
    type: 'image',
    metadata: { title: `Предмет #${id}` },
    created_at: new Date().toISOString(),
  };
}

/**
 * Создание предмета (при ручном вызове)
 */
export async function createItemReal(input: CreateItemInput): Promise<Item> {
  const item: Item = {
    id: Date.now(),
    creator_id: '',
    owner_id: '',
    s3_id: `s3-${Date.now()}`,
    type: input.type,
    metadata: {
      title: input.name,
      description: input.description,
      ...(input.metadata || {}),
    },
    created_at: new Date().toISOString(),
  };

  return item;
}
