import { useEffect, useState, useRef } from 'react';
import { subscribePublic, type PublicStreamEvent } from '~/api/realtime';
import type { StreamStatus } from './connection-status';

export function usePublicStream(onEvent?: (event: PublicStreamEvent) => void) {
  const [lastEvent, setLastEvent] = useState<PublicStreamEvent | null>(null);
  const [status, setStatus] = useState<StreamStatus>({
    state: 'connected',
    lastEventAt: null,
  });

  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;

  useEffect(() => {
    const unsubscribe = subscribePublic((event) => {
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
  }, []);

  return { lastEvent, status };
}
