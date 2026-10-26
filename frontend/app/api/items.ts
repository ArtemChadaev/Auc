import {
  listUserItemsReal,
  createItemReal,
  getItemReal,
  getItemDownloadUrlReal,
  giftItemReal,
  saveUserItemLocally,
} from './real/items';
import type { Item, CreateItemInput, DownloadUrlResponse } from './types';

const USE_MOCKS = false;

export async function listUserItems(): Promise<Item[]> {
  return listUserItemsReal();
}

export async function getItem(id: number): Promise<Item> {
  return getItemReal(id);
}

export async function createItem(input: CreateItemInput): Promise<Item> {
  return createItemReal(input);
}

/**
 * GET /api/item/{id}/download
 * Получение Presigned URL на скачивание файла (с проверкой суточного лимита 1 раз в день)
 */
export async function getItemDownloadUrl(id: number): Promise<DownloadUrlResponse> {
  return getItemDownloadUrlReal(id);
}

/**
 * POST /api/item/gift
 * Передает предмет новому владельцу с сохранением истории передачи
 */
export async function giftItem(
  itemId: number,
  newOwner: string,
  description?: string
): Promise<void> {
  return giftItemReal(itemId, newOwner, description);
}

export { saveUserItemLocally };
