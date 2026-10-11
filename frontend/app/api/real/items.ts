import type { Item, CreateItemInput } from '~/api/types';

// TODO(real-api): GET /api/user/items (MISSING IN GO)
export async function listUserItemsReal(): Promise<Item[]> {
  throw new Error('Real API: listUserItems is not implemented');
}

// TODO(real-api): POST /api/upload/url + POST /api/upload/confirm
export async function createItemReal(_input: CreateItemInput): Promise<Item> {
  throw new Error('Real API: createItem is not implemented');
}

// TODO(real-api): GET /api/item/{id}
export async function getItemReal(_id: number): Promise<Item> {
  throw new Error('Real API: getItem is not implemented');
}

// TODO(real-api): GET /api/item/{id}/download
export async function getItemDownloadUrlReal(_id: number): Promise<{ url: string }> {
  throw new Error('Real API: getItemDownloadUrl is not implemented');
}
