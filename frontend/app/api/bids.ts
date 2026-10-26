import { listBidsReal, placeBidReal } from './real/bids';
import type { Bid, PlaceBidResult } from './types';

const USE_MOCKS = false;

export async function listBids(lotId: number): Promise<Bid[]> {
  return listBidsReal(lotId);
}

export async function placeBid(
  lotId: number,
  amount: number,
  idempotencyKey?: string
): Promise<PlaceBidResult> {
  return placeBidReal(lotId, amount, idempotencyKey);
}
