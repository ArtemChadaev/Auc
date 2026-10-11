import {
  realtimeSimulator,
  type LotStreamEvent,
  type PublicStreamEvent,
  type UserStreamEvent,
} from './mocks/realtime-sim';

export type { LotStreamEvent, PublicStreamEvent, UserStreamEvent };

const USE_MOCKS = true;

export function subscribeLot(lotId: number, handler: (event: LotStreamEvent) => void): () => void {
  if (USE_MOCKS) {
    return realtimeSimulator.subscribeLot(lotId, handler);
  }
  // TODO(real-api): EventSource /api/lot/{id}/stream
  throw new Error('Real realtime stream not implemented');
}

export function subscribePublic(handler: (event: PublicStreamEvent) => void): () => void {
  if (USE_MOCKS) {
    return realtimeSimulator.subscribePublic(handler);
  }
  // TODO(real-api): EventSource /api/stream/public
  throw new Error('Real public stream not implemented');
}

export function subscribeUser(handler: (event: UserStreamEvent) => void): () => void {
  if (USE_MOCKS) {
    return realtimeSimulator.subscribeUser(handler);
  }
  // TODO(real-api): EventSource /api/stream/user
  throw new Error('Real user stream not implemented');
}
