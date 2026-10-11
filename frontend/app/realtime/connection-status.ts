export type ConnectionState = 'connected' | 'connecting' | 'disconnected';

export interface StreamStatus {
  state: ConnectionState;
  lastEventAt: string | null;
}
