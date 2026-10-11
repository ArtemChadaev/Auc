import { mockDb } from './mocks/db';
import { simulateLatency } from './mocks/latency';
import {
  listUserItemsReal,
  createItemReal,
  getItemReal,
  getItemDownloadUrlReal,
} from './real/items';
import type { Item, CreateItemInput, DownloadUrlResponse } from './types';

const USE_MOCKS = true;

export async function listUserItems(): Promise<Item[]> {
  if (USE_MOCKS) {
    await simulateLatency();
    return mockDb.listUserItems();
  }
  return listUserItemsReal();
}

export async function getItem(id: number): Promise<Item> {
  if (USE_MOCKS) {
    await simulateLatency(100);
    return mockDb.getItem(id);
  }
  return getItemReal(id);
}

export async function createItem(input: CreateItemInput): Promise<Item> {
  if (USE_MOCKS) {
    await simulateLatency();
    return mockDb.createItem(input);
  }
  return createItemReal(input);
}

/**
 * GET /api/item/{id}/download
 * Получение Presigned URL на скачивание файла (с проверкой суточного лимита 1 раз в день)
 */
export async function getItemDownloadUrl(id: number): Promise<DownloadUrlResponse> {
  if (USE_MOCKS) {
    await simulateLatency(150);
    return mockDb.getItemDownloadUrl(id);
  }
  return getItemDownloadUrlReal(id);
}
