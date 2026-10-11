/**
 * Симулятор realtime SSE потоков.
 * Работает полностью локально в браузере, эмулируя конкурентные ставки,
 * срабатывание anti-sniping, завершение лотов и персональные уведомления о возврате hold.
 */

import { mockDb } from './db';
import type { Lot, Bid, WinnerInfo } from '~/api/types';

export interface LotStreamEvent {
  type: 'bid_placed' | 'time_extended' | 'lot_finished' | 'lot_cancelled';
  lot_id: number;
  data: {
    bid?: Bid;
    current_bid?: number;
    bids_count?: number;
    ends_at?: string;
    reason?: string;
    winner?: WinnerInfo | null;
    finish_amount?: number;
    finished_at?: string;
  };
}

export interface PublicStreamEvent {
  type: 'lot_updated' | 'lot_started' | 'lot_finished';
  lot_id: number;
  data: {
    current_bid?: number;
    bids_count?: number;
    ends_at?: string;
    winner?: WinnerInfo | null;
    finish_amount?: number;
  };
}

export interface UserStreamEvent {
  type: 'balance_changed' | 'outbid_notification' | 'lot_won';
  data: {
    balance?: number;
    hold?: number;
    available?: number;
    lot_id?: number;
    lot_title?: string;
    your_bid?: number;
    new_bid?: number;
  };
}

type LotHandler = (event: LotStreamEvent) => void;
type PublicHandler = (event: PublicStreamEvent) => void;
type UserHandler = (event: UserStreamEvent) => void;

class RealtimeSimulator {
  private lotListeners: Map<number, Set<LotHandler>> = new Map();
  private publicListeners: Set<PublicHandler> = new Set();
  private userListeners: Set<UserHandler> = new Set();
  private timer: NodeJS.Timeout | null = null;
  private isRunning = false;

  constructor() {
    this.startSimulation();
  }

  public subscribeLot(lotId: number, handler: LotHandler): () => void {
    if (!this.lotListeners.has(lotId)) {
      this.lotListeners.set(lotId, new Set());
    }
    this.lotListeners.get(lotId)!.add(handler);

    return () => {
      const set = this.lotListeners.get(lotId);
      if (set) {
        set.delete(handler);
        if (set.size === 0) {
          this.lotListeners.delete(lotId);
        }
      }
    };
  }

  public subscribePublic(handler: PublicHandler): () => void {
    this.publicListeners.add(handler);
    return () => {
      this.publicListeners.delete(handler);
    };
  }

  public subscribeUser(handler: UserHandler): () => void {
    this.userListeners.add(handler);
    return () => {
      this.userListeners.delete(handler);
    };
  }

  // Публикация событий наружу
  public dispatchLotEvent(event: LotStreamEvent) {
    const set = this.lotListeners.get(event.lot_id);
    if (set) {
      set.forEach((fn) => fn(event));
    }
  }

  public dispatchPublicEvent(event: PublicStreamEvent) {
    this.publicListeners.forEach((fn) => fn(event));
  }

  public dispatchUserEvent(event: UserStreamEvent) {
    this.userListeners.forEach((fn) => fn(event));
  }

  private startSimulation() {
    if (this.isRunning) return;
    this.isRunning = true;

    // Циклический тик каждые 2 секунды (проверка завершения лотов + случайные ставки)
    this.timer = setInterval(() => {
      this.checkLotsStatus();
      this.simulateCompetitorActivity();
    }, 2500);
  }

  /**
   * Проверка окончания времени лотов
   */
  private checkLotsStatus() {
    const { items } = mockDb.listLots({ status: 'active', limit: 50 });
    const now = Date.now();

    for (const lot of items) {
      if (lot.ends_at) {
        const endsMs = new Date(lot.ends_at).getTime();
        if (now >= endsMs) {
          // Завершаем лот!
          this.finishLot(lot);
        }
      }
    }
  }

  private finishLot(lot: Lot) {
    const bids = mockDb.listBids(lot.id);
    const winningBid = bids[0];

    lot.status = 'finished';
    lot.finished_at = new Date().toISOString();
    lot.finish_amount = winningBid ? winningBid.amount : null;
    lot.winner = winningBid
      ? {
          id: winningBid.user_id,
          name: winningBid.user_name,
        }
      : null;

    // Оповещаем подписчиков лота и каталога
    this.dispatchLotEvent({
      type: 'lot_finished',
      lot_id: lot.id,
      data: {
        winner: lot.winner,
        finish_amount: lot.finish_amount || undefined,
        finished_at: lot.finished_at,
      },
    });

    this.dispatchPublicEvent({
      type: 'lot_finished',
      lot_id: lot.id,
      data: {
        winner: lot.winner,
        finish_amount: lot.finish_amount || undefined,
      },
    });

    // Если выиграл текущий пользователь
    const currentUser = mockDb.getCurrentUser();
    if (winningBid && winningBid.user_id === currentUser.id) {
      this.dispatchUserEvent({
        type: 'lot_won',
        data: {
          lot_id: lot.id,
          lot_title: lot.title,
          your_bid: winningBid.amount,
        },
      });
    }
  }

  /**
   * Имитация действий других участников (конкурентные ставки)
   */
  private simulateCompetitorActivity() {
    // 30% вероятность ставки на одном из активных лотов в этот тик
    if (Math.random() > 0.35) return;

    const { items: activeLots } = mockDb.listLots({ status: 'active', limit: 10 });
    if (activeLots.length === 0) return;

    // Выбираем случайный лот (предпочтительно лоты с близким временем окончания)
    const lot = activeLots[Math.floor(Math.random() * activeLots.length)];
    const currentUser = mockDb.getCurrentUser();

    // Не ставим от чужого имени на свой лот, если продавец - другой бот, ставит другой бот
    const competitorId = lot.seller_id === 'user-alice' ? 'user-bob' : 'user-alice';
    const nextAmount = lot.current_bid + lot.min_step;

    try {
      const prevBids = mockDb.listBids(lot.id);
      const prevLeader = prevBids[0];

      // Ставим ставку
      const result = mockDb.placeBid(lot.id, nextAmount, competitorId);

      // Проверяем, было ли продление Anti-sniping
      const wasExtended =
        new Date(result.lot.ends_at!).getTime() > new Date(lot.ends_at!).getTime();

      // Шлем событие в стрим лота
      this.dispatchLotEvent({
        type: wasExtended ? 'time_extended' : 'bid_placed',
        lot_id: lot.id,
        data: {
          bid: result.bid,
          current_bid: result.lot.current_bid,
          bids_count: result.lot.bids_count,
          ends_at: result.lot.ends_at || undefined,
          reason: wasExtended ? 'anti_sniping' : undefined,
        },
      });

      // Шлем в публичный каталог
      this.dispatchPublicEvent({
        type: 'lot_updated',
        lot_id: lot.id,
        data: {
          current_bid: result.lot.current_bid,
          bids_count: result.lot.bids_count,
          ends_at: result.lot.ends_at || undefined,
        },
      });

      // Если предыдущим лидером был наш текущий пользователь — уведомляем о перебивании!
      if (prevLeader && prevLeader.user_id === currentUser.id) {
        const updatedMe = mockDb.getCurrentUser();
        this.dispatchUserEvent({
          type: 'outbid_notification',
          data: {
            lot_id: lot.id,
            lot_title: lot.title,
            your_bid: prevLeader.amount,
            new_bid: nextAmount,
          },
        });

        this.dispatchUserEvent({
          type: 'balance_changed',
          data: {
            balance: updatedMe.balance,
            hold: updatedMe.hold,
            available: updatedMe.balance - updatedMe.hold,
          },
        });
      }
    } catch {
      // Игнорируем ошибки симуляции
    }
  }
}

export const realtimeSimulator = new RealtimeSimulator();
