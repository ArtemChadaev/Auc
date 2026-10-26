import { listTransactionsReal, topUpDailyCoinsReal } from './real/profile';
import type { LedgerEntry, User } from './types';

const USE_MOCKS = false;

export async function listTransactions(): Promise<LedgerEntry[]> {
  return listTransactionsReal();
}

export async function topUpDailyCoins(): Promise<{ user: User; added: boolean }> {
  return topUpDailyCoinsReal();
}
