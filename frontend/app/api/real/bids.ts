import { api, getStoredToken, generateIdempotencyKey } from '~/api/client';
import { getLotReal } from './lots';
import { getCurrentUserReal } from './auth';
import type { Bid, PlaceBidResult } from '~/api/types';
import { recordParticipatingLot } from '~/lib/participating-lots';

/**
 * GET /api/bid/{id}/stream
 * Получает актуальный список ставок лота через чтение первоначального кадра SSE потока
 */
export async function listBidsReal(lotId: number): Promise<Bid[]> {
  try {
    const controller = new AbortController();
    const token = getStoredToken();
    const headers: Record<string, string> = {
      Accept: 'text/event-stream',
    };
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`/api/bid/${lotId}/stream`, {
      method: 'GET',
      headers,
      signal: controller.signal,
    });

    if (!response.ok || !response.body) {
      return [];
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { value, done } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');

      for (const line of lines) {
        if (line.startsWith('data:')) {
          const jsonStr = line.replace(/^data:\s*/, '').trim();
          if (jsonStr) {
            try {
              const rawBids = JSON.parse(jsonStr);
              controller.abort(); // Первый кадр получен, закрываем стрим
              if (Array.isArray(rawBids)) {
                return rawBids.map((b: any, index: number) => ({
                  id: b.id || index + 1,
                  lot_id: b.lot_id || lotId,
                  user_id: b.user_id,
                  user_name: b.user_name || `Участник #${String(b.user_id).slice(0, 5)}`,
                  amount: Number(b.amount),
                  idempotency_key: b.idempotency_key || '',
                  created_at: b.created_at || new Date().toISOString(),
                }));
              }
            } catch {
              // Игнорируем ошибку парсинга промежуточных чанков
            }
          }
        }
      }
    }
    return [];
  } catch {
    return [];
  }
}

/**
 * POST /api/bid/{id}
 * Размещает новую ставку авторизованного пользователя на активный лот аукциона
 */
export async function placeBidReal(
  lotId: number,
  amount: number,
  customIdempotencyKey?: string
): Promise<PlaceBidResult> {
  const roundedAmount = Math.round(amount);
  const idempotencyKey =
    customIdempotencyKey || generateIdempotencyKey(`bid-${lotId}-${roundedAmount}`);

  await api.post(
    `/api/bid/${lotId}`,
    {
      amount: roundedAmount,
    },
    {
      headers: {
        'Idempotency-Key': idempotencyKey,
      },
    }
  );

  // Получаем обновленный лот и пользователя
  const [updatedLot, user] = await Promise.all([
    getLotReal(lotId).catch(() => null),
    getCurrentUserReal().catch(() => null),
  ]);

  if (user?.id) {
    recordParticipatingLot(lotId, user.id);
  }

  const newBid: Bid = {
    id: Date.now(),
    lot_id: lotId,
    user_id: user?.id || '',
    user_name: user?.name || 'Вы',
    amount: roundedAmount,
    idempotency_key: idempotencyKey,
    created_at: new Date().toISOString(),
  };

  return {
    bid: newBid,
    lot: updatedLot || ({} as any),
    user_balance: user?.balance ?? 0,
    user_hold: user?.hold ?? 0,
  };
}
