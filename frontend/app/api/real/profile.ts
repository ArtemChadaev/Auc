import { getCurrentUserReal, getFreeCoinReal } from './auth';
import { getMyLotsReal } from './lots';
import type { LedgerEntry, User } from '~/api/types';

const STORED_LEDGER_KEY = 'auc_user_ledger';

export function saveTransactionLocally(entry: LedgerEntry): void {
  if (typeof window === 'undefined') return;
  try {
    const raw = localStorage.getItem(STORED_LEDGER_KEY);
    const list: LedgerEntry[] = raw ? JSON.parse(raw) : [];
    list.unshift(entry);
    localStorage.setItem(STORED_LEDGER_KEY, JSON.stringify(list));
  } catch {
    // Ignore storage quota
  }
}

/**
 * GET /api/user/transactions
 * Возвращает историю операций пользователя
 */
export async function listTransactionsReal(): Promise<LedgerEntry[]> {
  const localTx: LedgerEntry[] = (() => {
    if (typeof window === 'undefined') return [];
    try {
      const raw = localStorage.getItem(STORED_LEDGER_KEY);
      return raw ? JSON.parse(raw) : [];
    } catch {
      return [];
    }
  })();

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

    const combined = [...localTx, ...dynamicEntries];
    const unique = Array.from(new Map(combined.map((e) => [e.id, e])).values());
    return unique.sort(
      (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    );
  } catch {
    return localTx;
  }
}

/**
 * Начисление бесплатных монет Coins (GET /api/user/me/freeCoin)
 */
export async function topUpDailyCoinsReal(): Promise<{ user: User; added: boolean }> {
  const user = await getFreeCoinReal();
  return { user, added: true };
}
