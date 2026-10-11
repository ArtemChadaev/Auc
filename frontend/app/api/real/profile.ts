import type { LedgerEntry, User } from '~/api/types';

// TODO(real-api): GET /api/user/transactions (MISSING IN GO)
export async function listTransactionsReal(): Promise<LedgerEntry[]> {
  throw new Error('Real API: listTransactions is not implemented');
}

// TODO(real-api): POST /api/user/top-up (MISSING IN GO)
export async function topUpDailyCoinsReal(): Promise<{ user: User; added: boolean }> {
  throw new Error('Real API: topUpDailyCoins is not implemented');
}
