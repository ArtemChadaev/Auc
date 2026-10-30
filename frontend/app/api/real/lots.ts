import { api, generateIdempotencyKey } from '~/api/client';
import { resolveAvatarUrl, S3_PHOTOS_BASE } from '~/api/real/auth';
import type { Lot, LotFilter, LotListResponse, CreateLotInput, ItemType } from '~/api/types';

export { S3_PHOTOS_BASE };

let cachedPhotoPrefix: string | null = null;

/**
 * GET /api/lot/photo/prefix
 * Получает базовый URL префикс для скачивания фотографий по UUID
 */
export async function getPhotoPrefixReal(): Promise<string> {
  if (cachedPhotoPrefix) return cachedPhotoPrefix;
  try {
    const res = await api.get<{ prefix: string }>('/api/lot/photo/prefix');
    if (res?.prefix) {
      cachedPhotoPrefix = res.prefix.replace(/\/$/, '');
      return cachedPhotoPrefix;
    }
  } catch {
    // fallback quietly к стандартному URL
  }
  return S3_PHOTOS_BASE;
}

export function setPhotoPrefix(prefix: string) {
  cachedPhotoPrefix = prefix.replace(/\/$/, '');
}

if (typeof window !== 'undefined') {
  getPhotoPrefixReal().catch(() => {});
}

export function resolvePhotoUrl(photo?: string | null): string {
  if (!photo) return '/placeholder.svg';
  if (photo.startsWith('http://') || photo.startsWith('https://') || photo.startsWith('data:')) {
    return photo;
  }
  const cleanId = photo.replace(/^\/api\/lot\/photo\//, '').replace(/^.*\//, '').trim();
  if (!cleanId) return '/placeholder.svg';
  const base = cachedPhotoPrefix || S3_PHOTOS_BASE;
  return `${base}/${cleanId}`;
}

function mapCategoryToItemType(category: string): ItemType {
  const cat = (category || '').toLowerCase();
  if (cat.includes('3d') || cat.includes('модел')) return '3d';
  if (cat.includes('аудио') || cat.includes('audio') || cat.includes('звук')) return 'audio';
  if (cat.includes('видео') || cat.includes('video')) return 'video';
  if (cat.includes('архив') || cat.includes('archive')) return 'archive';
  if (cat.includes('документ') || cat.includes('doc')) return 'document';
  return 'image';
}

function normalizeLot(raw: any): Lot {
  if (!raw) return raw;

  const id = Number(raw.id);
  const startAmount = Number(raw.start_amount ?? 0);
  const minStep = Number(raw.min_step ?? 50);
  const finishAmount = raw.finish_amount !== undefined && raw.finish_amount !== null
    ? Number(raw.finish_amount)
    : null;

  const rawDesc = raw.description || {};
  const descriptionType: ItemType = rawDesc.type || 'image';

  // Извлекаем чистые UUID из images или photo
  const rawList = Array.isArray(rawDesc.images) && rawDesc.images.length > 0
    ? rawDesc.images
    : Array.isArray(rawDesc.photo)
    ? rawDesc.photo
    : [];

  const rawImages: string[] = rawList
    .filter((p: any) => typeof p === 'string' && p.trim() !== '')
    .map((p: string) => p.replace(/^\/api\/lot\/photo\//, '').replace(/^.*\//, '').trim())
    .filter(Boolean);

  const rawPhotos: string[] = rawImages.map((p: string) => resolvePhotoUrl(p));
  const previewPhoto = rawPhotos[0] || resolvePhotoUrl(raw.item_summary?.preview) || '/placeholder.svg';

  return {
    id,
    seller_id: raw.seller_id || '',
    seller: raw.seller ? {
      ...raw.seller,
      avatar_id: resolveAvatarUrl(raw.seller.avatar_id),
    } : {
      id: raw.seller_id || '',
      name: raw.seller?.name || 'Продавец',
      avatar_id: resolveAvatarUrl(raw.seller?.avatar_id),
      rating: 5.0,
    },
    item_id: Number(raw.item_id ?? 0),
    item_summary: raw.item_summary ? {
      ...raw.item_summary,
      id: Number(raw.item_summary.id ?? raw.item_id ?? 0),
      type: raw.item_summary.type || descriptionType,
      preview: resolvePhotoUrl(raw.item_summary.preview) || previewPhoto,
    } : {
      id: Number(raw.item_id ?? 0),
      type: descriptionType,
      preview: previewPhoto,
    },
    status: raw.status || 'active',
    category: raw.category || descriptionType,
    title: raw.title || 'Лот аукциона',
    description: {
      text: rawDesc.text || '',
      short_text: rawDesc.short_text || '',
      images: rawImages,
      photo: rawPhotos,
      media: rawPhotos.map((p: string, idx: number) => ({
        id: `photo-${idx}`,
        type: 'image' as const,
        order: idx + 1,
        caption: '',
        url: p,
      })),
      attributes: rawDesc.attributes || {},
      auto_activate: rawDesc.auto_activate ?? true,
      reject_reason: rawDesc.reject_reason || undefined,
      draft_ui: rawDesc.draft_ui || undefined,
    },
    start_amount: startAmount,
    current_bid: Number(raw.current_bid ?? (finishAmount || startAmount)),
    min_step: minStep,
    finish_amount: finishAmount,
    bids_count: Number(raw.bids_count ?? 0),
    winner: raw.winner || (raw.winner_id ? { id: raw.winner_id, name: 'Победитель' } : null),
    created_at: raw.created_at || new Date().toISOString(),
    started_at: raw.started_at || null,
    ends_at: raw.ends_at || null,
    finished_at: raw.finished_at || null,
  };
}

/**
 * POST /api/lot/photo
 * Загружает фотографию для лота в хранилище photos S3 и возвращает идентификатор (UUID) и прямую ссылку
 */
export async function uploadLotPhotoReal(file: File | Blob): Promise<{ url: string; id: string }> {
  const formData = new FormData();
  formData.append('file', file);
  const res = await api.post<{ url?: string; direct_url?: string; id?: string; uuid?: string }>('/api/lot/photo', formData);
  const photoId = res.id || res.uuid || '';
  const prefix = await getPhotoPrefixReal();
  const direct = res.direct_url || (photoId ? `${prefix}/${photoId}` : res.url || '');
  return {
    id: photoId,
    url: direct,
  };
}

/**
 * GET /api/lot/
 * Возвращает список лотов аукциона со статусами active и finished
 */
export async function listLotsReal(filter: LotFilter = {}): Promise<LotListResponse> {
  const rawLots = await api.get<any[]>('/api/lot/');
  const list = Array.isArray(rawLots) ? rawLots.map(normalizeLot) : [];

  let filtered = [...list];

  // Фильтрация по статусу
  if (filter.status === 'upcoming') {
    const nowMs = Date.now();
    filtered = filtered.filter((l) => {
      const startMs = l.started_at ? new Date(l.started_at).getTime() : 0;
      return startMs > nowMs && l.status !== 'finished' && l.status !== 'cancelled';
    });
  } else if (filter.status === 'active') {
    const nowMs = Date.now();
    filtered = filtered.filter((l) => {
      const startMs = l.started_at ? new Date(l.started_at).getTime() : 0;
      return l.status === 'active' && (startMs === 0 || startMs <= nowMs);
    });
  } else if (filter.status) {
    filtered = filtered.filter((l) => l.status === filter.status);
  }

  // Фильтрация по поисковому запросу
  if (filter.query) {
    const q = filter.query.toLowerCase();
    filtered = filtered.filter(
      (l) =>
        l.title.toLowerCase().includes(q) ||
        l.category.toLowerCase().includes(q) ||
        l.description.short_text?.toLowerCase().includes(q) ||
        l.description.text.toLowerCase().includes(q)
    );
  }

  // Фильтрация по категории
  if (filter.category) {
    const cat = filter.category.toLowerCase();
    filtered = filtered.filter(
      (l) =>
        l.category.toLowerCase() === cat ||
        l.item_summary?.type.toLowerCase() === cat ||
        l.description.text.toLowerCase().includes(cat)
    );
  }

  // Фильтрация по цене
  if (filter.min_price !== undefined) {
    filtered = filtered.filter((l) => l.current_bid >= filter.min_price!);
  }
  if (filter.max_price !== undefined) {
    filtered = filtered.filter((l) => l.current_bid <= filter.max_price!);
  }

  // Сортировка
  if (filter.sort === 'price_asc') {
    filtered.sort((a, b) => a.current_bid - b.current_bid);
  } else if (filter.sort === 'price_desc') {
    filtered.sort((a, b) => b.current_bid - a.current_bid);
  } else if (filter.sort === 'popular') {
    filtered.sort((a, b) => b.bids_count - a.bids_count);
  } else if (filter.sort === 'ending_soon') {
    filtered.sort((a, b) => {
      const timeA = a.ends_at ? new Date(a.ends_at).getTime() : Infinity;
      const timeB = b.ends_at ? new Date(b.ends_at).getTime() : Infinity;
      return timeA - timeB;
    });
  } else {
    // newest
    filtered.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
  }

  const total = filtered.length;
  const page = filter.page || 1;
  const limit = filter.limit || 6;
  const total_pages = Math.ceil(total / limit) || 1;
  const offset = (page - 1) * limit;
  const items = filtered.slice(offset, offset + limit);

  return {
    items,
    total,
    page,
    limit,
    total_pages,
  };
}

const LOTS_CACHE = new Map<number, Lot>();

export function cacheLot(lot: Lot): void {
  if (lot?.id) {
    LOTS_CACHE.set(Number(lot.id), lot);
  }
}

/**
 * GET /api/lot/{id}
 * Возвращает информацию о лоте по его числовому идентификатору
 */
export async function getLotReal(id: number): Promise<Lot> {
  const numId = Number(id);
  try {
    const raw = await api.get<any>(`/api/lot/${numId}`);
    const normalized = normalizeLot(raw);
    cacheLot(normalized);
    return normalized;
  } catch (err: any) {
    // 1. Проверяем локальный кеш только что созданных/измененных лотов
    if (LOTS_CACHE.has(numId)) {
      return LOTS_CACHE.get(numId)!;
    }
    // 2. Публичный GET /api/lot/{id} возвращает 404 для лотов со статусом draft и pending_review
    // Для продавца лот доступен через личный эндпоинт GET /api/lot/my
    if (err?.status === 404 || err?.message?.toLowerCase().includes('not found')) {
      try {
        const myLots = await getMyLotsReal();
        const found = myLots.find((l) => l.id === numId);
        if (found) {
          cacheLot(found);
          return found;
        }
      } catch {
        // Игнорируем сетевые сбои getMyLots
      }
    }
    throw err;
  }
}

/**
 * POST /api/lot/
 * Создает лот на аукционе для принадлежащего пользователю предмета
 */
export async function createLotReal(input: CreateLotInput): Promise<Lot> {
  const itemType = mapCategoryToItemType(input.category);

  // Валидация кратностей по контракту Go:
  // start_amount >= 100 и кратно 100
  // min_step от 50 до 500 и кратно 50
  const startAmount = Math.max(100, Math.round(input.start_amount / 100) * 100);
  const minStep = Math.min(500, Math.max(50, Math.round(input.min_step / 50) * 50));

  const rawList = input.images || input.photos || [];
  const cleanUUIDs = rawList
    .filter((p: any) => typeof p === 'string' && p.trim() !== '')
    .map((p: string) => p.replace(/^\/api\/lot\/photo\//, '').replace(/^.*\//, '').trim())
    .filter(Boolean);

  const body: any = {
    item_id: Number(input.item_id),
    title: input.title.trim(),
    start_amount: startAmount,
    min_step: minStep,
    status: input.status || 'pending_review',
    description: {
      type: itemType,
      text: input.text?.trim() || input.title.trim(),
      short_text: input.short_text?.trim() || '',
      images: cleanUUIDs,
      photo: cleanUUIDs,
      tags: [],
      auto_activate: input.auto_activate ?? true,
      draft_ui: input.draft_ui,
    },
  };

  if (input.started_at && input.ends_at) {
    body.started_at = input.started_at;
    body.ends_at = input.ends_at;
  }

  const idempotencyKey = generateIdempotencyKey(`lot-create-${input.item_id}`);
  const raw = await api.post<any>('/api/lot/', body, { headers: { 'Idempotency-Key': idempotencyKey } });
  const normalized = normalizeLot(raw);
  cacheLot(normalized);
  return normalized;
}

/**
 * PUT /api/lot/{id}
 * Обновляет параметры лота продавцом по его ID
 */
export async function updateLotReal(id: number, input: Partial<CreateLotInput>): Promise<Lot> {
  const itemType = input.category ? mapCategoryToItemType(input.category) : 'image';
  const startAmount = input.start_amount
    ? Math.max(100, Math.round(input.start_amount / 100) * 100)
    : 100;
  const minStep = input.min_step
    ? Math.min(500, Math.max(50, Math.round(input.min_step / 50) * 50))
    : 50;

  const rawList = input.images || input.photos || [];
  const cleanUUIDs = rawList
    .filter((p: any) => typeof p === 'string' && p.trim() !== '')
    .map((p: string) => p.replace(/^\/api\/lot\/photo\//, '').replace(/^.*\//, '').trim())
    .filter(Boolean);

  const body: any = {
    item_id: input.item_id ? Number(input.item_id) : undefined,
    title: input.title?.trim() || 'Лот',
    start_amount: startAmount,
    min_step: minStep,
    status: input.status || 'draft',
    description: {
      type: itemType,
      text: input.text?.trim() || input.title?.trim() || '',
      short_text: input.short_text?.trim() || '',
      images: cleanUUIDs,
      photo: cleanUUIDs,
      tags: [],
      auto_activate: input.auto_activate ?? true,
      draft_ui: input.draft_ui,
    },
  };

  if (input.started_at && input.ends_at) {
    body.started_at = input.started_at;
    body.ends_at = input.ends_at;
  }

  const idempotencyKey = generateIdempotencyKey(`lot-update-${id}`);
  const raw = await api.put<any>(`/api/lot/${id}`, body, { headers: { 'Idempotency-Key': idempotencyKey } });
  const normalized = normalizeLot(raw);
  cacheLot(normalized);
  return normalized;
}

/**
 * DELETE /api/lot/{id}
 * Отменяет лот продавца
 */
export async function cancelLotReal(id: number): Promise<void> {
  const idempotencyKey = generateIdempotencyKey(`lot-cancel-${id}`);
  await api.delete(`/api/lot/${id}`, { headers: { 'Idempotency-Key': idempotencyKey } });
}

/**
 * GET /api/lot/my
 * Возвращает все лоты, созданные текущим пользователем
 */
export async function getMyLotsReal(): Promise<Lot[]> {
  const rawLots = await api.get<any[]>('/api/lot/my');
  return Array.isArray(rawLots) ? rawLots.map(normalizeLot) : [];
}

/**
 * POST /api/lot/{id}/activate
 * Переводит лот продавца со статусом approved в статус active для старта торгов
 */
export async function activateLotReal(
  id: number,
  startedAt?: string,
  endsAt?: string
): Promise<Lot> {
  const body: any = {};
  if (startedAt) body.started_at = startedAt;
  if (endsAt) body.ends_at = endsAt;
  const idempotencyKey = generateIdempotencyKey(`lot-activate-${id}`);
  const raw = await api.post<any>(`/api/lot/${id}/activate`, body, { headers: { 'Idempotency-Key': idempotencyKey } });
  const normalized = normalizeLot(raw);
  cacheLot(normalized);
  return normalized;
}
