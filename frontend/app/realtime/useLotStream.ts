import { useEffect, useState, useRef } from 'react';
import { subscribeLot, type LotStreamEvent } from '~/api/realtime';
import type { StreamStatus } from './connection-status';

export function useLotStream(lotId: number | null, onEvent?: (event: LotStreamEvent) => void) {
  const [lastEvent, setLastEvent] = useState<LotStreamEvent | null>(null);
  const [status, setStatus] = useState<StreamStatus>({
    state: 'connecting',
    lastEventAt: null,
  });

  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;

  useEffect(() => {
    if (!lotId) return;

    setStatus({ state: 'connected', lastEventAt: null });

    const unsubscribe = subscribeLot(lotId, (event) => {
      setLastEvent(event);
      setStatus({
        state: 'connected',
        lastEventAt: new Date().toISOString(),
      });
      if (onEventRef.current) {
        onEventRef.current(event);
      }
    });

    return () => {
      unsubscribe();
      setStatus({ state: 'disconnected', lastEventAt: null });
    };
  }, [lotId]);

  return { lastEvent, status };
}
