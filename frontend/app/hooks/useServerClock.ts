import { useState } from 'react';

/**
 * Хук для отслеживания серверного времени и смещения локальных часов.
 */
export function useServerClock() {
  const [offsetMs] = useState(0);

  const getNow = () => Date.now() + offsetMs;

  return {
    offsetMs,
    getNow,
  };
}
