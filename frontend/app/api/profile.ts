import { mockDb } from './mocks/db';
import { simulateLatency } from './mocks/latency';
import { listTransactionsReal, topUpDailyCoinsReal } from './real/profile';
import type { LedgerEntry, User } from './types';

const USE_MOCKS = true;

export async function listTransactions(): Promise<LedgerEntry[]> {
  if (USE_MOCKS) {
    await simulateLatency();
    return mockDb.listTransactions();
  }
  return listTransactionsReal();
}

export async function topUpDailyCoins(): Promise<{ user: User; added: boolean }> {
  if (USE_MOCKS) {
    await simulateLatency();
    return mockDb.topUpDailyCoins();
  }
  return topUpDailyCoinsReal();
}
