import {
  listLotsReal,
  getLotReal,
  createLotReal,
  updateLotReal,
  cancelLotReal,
  getMyLotsReal,
  activateLotReal,
  uploadLotPhotoReal,
  getPhotoPrefixReal,
  resolvePhotoUrl,
  cacheLot,
} from './real/lots';
import type { Lot, LotFilter, LotListResponse, CreateLotInput } from './types';

const USE_MOCKS = false;

export async function uploadLotPhoto(file: File | Blob): Promise<{ url: string; id: string }> {
  return uploadLotPhotoReal(file);
}

export async function getPhotoPrefix(): Promise<string> {
  return getPhotoPrefixReal();
}

export { resolvePhotoUrl };

export async function listLots(filter: LotFilter = {}): Promise<LotListResponse> {
  return listLotsReal(filter);
}

export async function getLot(id: number): Promise<Lot> {
  return getLotReal(id);
}

export async function createLot(input: CreateLotInput): Promise<Lot> {
  return createLotReal(input);
}

export async function updateLot(id: number, input: Partial<CreateLotInput>): Promise<Lot> {
  return updateLotReal(id, input);
}

export async function cancelLot(id: number): Promise<void> {
  return cancelLotReal(id);
}

export async function getMyLots(): Promise<Lot[]> {
  return getMyLotsReal();
}

export async function activateLot(
  id: number,
  startedAt?: string,
  endsAt?: string
): Promise<Lot> {
  return activateLotReal(id, startedAt, endsAt);
}

export { cacheLot } from './real/lots';
