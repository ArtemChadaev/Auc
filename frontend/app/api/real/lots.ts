import type { Lot, LotFilter, LotListResponse, CreateLotInput } from '~/api/types';

// TODO(real-api): GET /api/lot/
export async function listLotsReal(_filter: LotFilter): Promise<LotListResponse> {
  throw new Error('Real API: listLots is not implemented');
}

// TODO(real-api): GET /api/lot/{id}
export async function getLotReal(_id: number): Promise<Lot> {
  throw new Error('Real API: getLot is not implemented');
}

// TODO(real-api): POST /api/lot/ (MISSING IN GO)
export async function createLotReal(_input: CreateLotInput): Promise<Lot> {
  throw new Error('Real API: createLot is not implemented');
}
