import { useEffect, useState, useRef } from 'react';
import { subscribeUser, type UserStreamEvent } from '~/api/realtime';
import type { StreamStatus } from './connection-status';

export function useUserStream(onEvent?: (event: UserStreamEvent) => void) {
  const [lastEvent, setLastEvent] = useState<UserStreamEvent | null>(null);
  const [status, setStatus] = useState<StreamStatus>({
    state: 'connected',
    lastEventAt: null,
  });

  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;

  useEffect(() => {
    const unsubscribe = subscribeUser((event) => {
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
