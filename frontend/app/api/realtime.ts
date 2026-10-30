import type { LotStreamEvent, PublicStreamEvent, UserStreamEvent } from './mocks/realtime-sim';
import { listLotsReal } from './real/lots';
import type { Lot } from './types';
import { getCurrentUserReal } from './real/auth';
import { getStoredToken } from './client';

export type { LotStreamEvent, PublicStreamEvent, UserStreamEvent };

const USE_MOCKS = false;

/**
 * SSE подписка на поток ставок конкретного лота через GET /api/bid/{id}/stream
 * Использует fetch с ReadableStream для безопасной передачи Authorization Bearer заголовка
 */
export function subscribeLot(lotId: number, handler: (event: LotStreamEvent) => void): () => void {
  if (typeof window === 'undefined') {
    return () => {};
  }

  const controller = new AbortController();
  let active = true;

  const startStream = async () => {
    if (!active) return;

    const token = getStoredToken();
    const headers: Record<string, string> = {
      Accept: 'text/event-stream',
    };
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    try {
      const response = await fetch(`/api/bid/${lotId}/stream`, {
        method: 'GET',
        headers,
        signal: controller.signal,
      });

      if (!response.ok || !response.body) {
        if (active) {
          setTimeout(startStream, 5000);
        }
        return;
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';

      while (active) {
        const { value, done } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (line.startsWith('data:')) {
            const jsonStr = line.replace(/^data:\s*/, '').trim();
            if (jsonStr) {
              try {
                const raw = JSON.parse(jsonStr);
                if (Array.isArray(raw)) {
                  const latest = raw[0];
                  handler({
                    type: 'bid_placed',
                    lot_id: lotId,
                    data: {
                      bid: latest
                        ? {
                            id: latest.id || Date.now(),
                            lot_id: lotId,
                            user_id: latest.user_id,
                            user_name: latest.user_name || `Участник #${String(latest.user_id).slice(0, 5)}`,
                            amount: Number(latest.amount),
                            idempotency_key: latest.idempotency_key || '',
                            created_at: latest.created_at || new Date().toISOString(),
                          }
                        : undefined,
                      current_bid: latest ? Number(latest.amount) : undefined,
                      bids_count: raw.length,
                    },
                  });
                }
              } catch {
                // Ignore parse errors on partial chunks
              }
            }
          }
        }
      }
    } catch {
      if (active) {
        setTimeout(startStream, 5000);
      }
    }
  };

  startStream();

  return () => {
    active = false;
    controller.abort();
  };
}

// -------------------------------------------------------------
// Мультиплексированный синглтон-поллер для публичных обновлений лотов
// -------------------------------------------------------------
const PUBLIC_POLL_INTERVAL_MS = 15000; // 15 секунд вместо 4 сек
const publicSubscribers = new Set<(event: PublicStreamEvent) => void>();
let publicTimer: ReturnType<typeof setInterval> | null = null;
let lastKnownBids: Record<number, number> = {};

async function pollPublicOnce() {
  if (publicSubscribers.size === 0) return;
  // Не нагружаем сервер, если вкладка свернута или не активна
  if (typeof document !== 'undefined' && document.hidden) return;

  try {
    const res = await listLotsReal({ status: 'active', limit: 20 });
    res.items.forEach((lot: Lot) => {
      const prev = lastKnownBids[lot.id];
      if (prev !== undefined && lot.current_bid !== prev) {
        const ev: PublicStreamEvent = {
          type: 'lot_updated',
          lot_id: lot.id,
          data: {
            current_bid: lot.current_bid,
            bids_count: lot.bids_count,
            ends_at: lot.ends_at || undefined,
          },
        };
        publicSubscribers.forEach((fn) => fn(ev));
      }
      lastKnownBids[lot.id] = lot.current_bid;
    });
  } catch {
    // Ignore network errors in polling
  }
}

/**
 * Подписка на публичные обновления лотов (единый синглтон с интервалом 15 секунд)
 */
export function subscribePublic(handler: (event: PublicStreamEvent) => void): () => void {
  if (typeof window === 'undefined') {
    return () => {};
  }

  publicSubscribers.add(handler);

  if (!publicTimer) {
    publicTimer = setInterval(pollPublicOnce, PUBLIC_POLL_INTERVAL_MS);
  }

  return () => {
    publicSubscribers.delete(handler);
    if (publicSubscribers.size === 0 && publicTimer) {
      clearInterval(publicTimer);
      publicTimer = null;
    }
  };
}

// -------------------------------------------------------------
// Мультиплексированный синглтон-поллер для обновлений баланса пользователя
// -------------------------------------------------------------
const USER_POLL_INTERVAL_MS = 20000; // 20 секунд вместо 5 сек
const userSubscribers = new Set<(event: UserStreamEvent) => void>();
let userTimer: ReturnType<typeof setInterval> | null = null;
let lastBalance: number | null = null;
let lastHold: number | null = null;

async function pollUserOnce() {
  if (userSubscribers.size === 0) return;
  if (typeof document !== 'undefined' && document.hidden) return;

  const token = getStoredToken();
  if (!token) return; // Не опрашиваем, если не авторизован

  try {
    const u = await getCurrentUserReal();
    if (lastBalance !== null && (u.balance !== lastBalance || u.hold !== lastHold)) {
      const ev: UserStreamEvent = {
        type: 'balance_changed',
        data: {
          balance: u.balance,
          hold: u.hold,
          available: u.balance - u.hold,
        },
      };
      userSubscribers.forEach((fn) => fn(ev));
    }
    lastBalance = u.balance;
    lastHold = u.hold;
  } catch {
    // Ignore when not logged in
  }
}

/**
 * Подписка на персональные события пользователя (единый синглтон с интервалом 20 секунд)
 */
export function subscribeUser(handler: (event: UserStreamEvent) => void): () => void {
  if (typeof window === 'undefined') {
    return () => {};
  }

  userSubscribers.add(handler);

  if (!userTimer) {
    userTimer = setInterval(pollUserOnce, USER_POLL_INTERVAL_MS);
  }

  return () => {
    userSubscribers.delete(handler);
    if (userSubscribers.size === 0 && userTimer) {
      clearInterval(userTimer);
      userTimer = null;
    }
  };
}
