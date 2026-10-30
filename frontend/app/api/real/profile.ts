import { api } from '../client';
import { getCurrentUserReal, getFreeCoinReal } from './auth';
import { getMyLotsReal } from './lots';
import type { LedgerEntry, User } from '~/api/types';

/**
 * GET /api/user/transactions
 * Возвращает историю операций пользователя с сервера
 */
export async function listTransactionsReal(): Promise<LedgerEntry[]> {
  try {
    const serverEntries = await api.get<LedgerEntry[]>('/api/user/transactions');
    if (Array.isArray(serverEntries)) {
      return serverEntries.map((e) => ({
        ...e,
        id: Number(e.id),
        amount: Number(e.amount),
        balance_after: Number(e.balance_after),
        lot_id: e.lot_id ? Number(e.lot_id) : undefined,
      }));
    }
  } catch {
    // fallback if server fails
  }

  try {
    const myLots = await getMyLotsReal().catch(() => []);
    const dynamicEntries: LedgerEntry[] = [];
    myLots.forEach((l) => {
      if (l.status === 'finished' && l.finish_amount) {
        dynamicEntries.push({
          id: Number(l.id) * 1000 + 1,
          user_id: l.seller_id,
          amount: l.finish_amount,
          balance_after: 0,
          type: 'transfer_to_seller',
          lot_id: l.id,
          created_at: l.finished_at || l.created_at,
        });
      }
    });

    const unique = Array.from(new Map(dynamicEntries.map((e) => [e.id, e])).values());
    return unique.sort(
      (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    );
  } catch {
    return [];
  }
}

/**
 * Начисление бесплатных монет Coins (GET /api/user/me/freeCoin)
 */
export async function topUpDailyCoinsReal(): Promise<{ user: User; added: boolean }> {
  const user = await getFreeCoinReal();
  return { user, added: true };
}
