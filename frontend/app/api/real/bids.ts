import type { Bid, PlaceBidResult } from '~/api/types';

// TODO(real-api): GET /api/lot/{id}/bids (MISSING IN GO)
export async function listBidsReal(_lotId: number): Promise<Bid[]> {
  throw new Error('Real API: listBids is not implemented');
}

// TODO(real-api): POST /api/lot/{id}/bids (MISSING IN GO)
export async function placeBidReal(_lotId: number, _amount: number): Promise<PlaceBidResult> {
  throw new Error('Real API: placeBid is not implemented');
}
