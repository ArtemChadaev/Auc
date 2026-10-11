import { mockDb } from './mocks/db';
import { simulateLatency } from './mocks/latency';
import { listLotsReal, getLotReal, createLotReal } from './real/lots';
import type { Lot, LotFilter, LotListResponse, CreateLotInput } from './types';

const USE_MOCKS = true;

export async function listLots(filter: LotFilter = {}): Promise<LotListResponse> {
  if (USE_MOCKS) {
    await simulateLatency();
    return mockDb.listLots(filter);
  }
  return listLotsReal(filter);
}

export async function getLot(id: number): Promise<Lot> {
  if (USE_MOCKS) {
    await simulateLatency();
    return mockDb.getLot(id);
  }
  return getLotReal(id);
}

export async function createLot(input: CreateLotInput): Promise<Lot> {
  if (USE_MOCKS) {
    await simulateLatency();
    return mockDb.createLot(input);
  }
  return createLotReal(input);
}
