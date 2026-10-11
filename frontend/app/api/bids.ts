import { mockDb } from './mocks/db';
import { simulateLatency } from './mocks/latency';
import { listBidsReal, placeBidReal } from './real/bids';
import type { Bid, PlaceBidResult } from './types';

const USE_MOCKS = true;

export async function listBids(lotId: number): Promise<Bid[]> {
  if (USE_MOCKS) {
    await simulateLatency();
    return mockDb.listBids(lotId);
  }
  return listBidsReal(lotId);
}

export async function placeBid(lotId: number, amount: number): Promise<PlaceBidResult> {
  if (USE_MOCKS) {
    await simulateLatency();
    return mockDb.placeBid(lotId, amount);
  }
  return placeBidReal(lotId, amount);
}
