import { api, generateIdempotencyKey } from '~/api/client';
import { getMyLotsReal } from './lots';
import type { Item, CreateItemInput, DownloadUrlResponse, ItemType } from '~/api/types';

const STORED_ITEMS_KEY = 'auc_user_items';

export function saveUserItemLocally(item: Item): void {
  if (typeof window === 'undefined') return;
  try {
    const raw = localStorage.getItem(STORED_ITEMS_KEY);
    const list: Item[] = raw ? JSON.parse(raw) : [];
    const exists = list.some((i) => i.id === item.id);
    if (!exists) {
      list.unshift(item);
      localStorage.setItem(STORED_ITEMS_KEY, JSON.stringify(list));
    }
  } catch {
    // Ignore storage quota
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
 * Получение списка предметов текущего пользователя
 * (объединяет загруженные в S3 предметы и предметы из лотов продавца)
 */
export async function listUserItemsReal(): Promise<Item[]> {
  const localItems: Item[] = (() => {
    if (typeof window === 'undefined') return [];
    try {
      const raw = localStorage.getItem(STORED_ITEMS_KEY);
      return raw ? JSON.parse(raw) : [];
    } catch {
      return [];
    }
  })();

  try {
    const myLots = await getMyLotsReal().catch(() => []);
    const lotItems: Item[] = myLots
      .filter((lot) => lot.item_id)
      .map((lot) => ({
        id: lot.item_id,
        creator_id: lot.seller_id,
        owner_id: lot.seller_id,
        s3_id: `s3-${lot.item_id}`,
        type: (lot.item_summary?.type as ItemType) || ((lot.description as any)?.type as ItemType) || 'image',
        metadata: {
          title: lot.title,
          description: lot.description?.text,
        },
        created_at: lot.created_at,
      }));

    // Объединяем уникальные предметы
    const allMap = new Map<number, Item>();
    lotItems.forEach((i) => allMap.set(i.id, i));
    localItems.forEach((i) => allMap.set(i.id, i));

    return Array.from(allMap.values()).sort(
      (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    );
  } catch {
    return localItems;
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
 * Создание предмета (при ручном вызове сохраняет локально)
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

  saveUserItemLocally(item);
  return item;
}
