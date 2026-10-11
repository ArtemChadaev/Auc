import { useState, useEffect } from 'react';
import { calculateRemainingTime, type RemainingTime } from '~/lib/date';

export function useCountdown(targetIso: string | null | undefined, offsetMs = 0): RemainingTime {
  const [remaining, setRemaining] = useState<RemainingTime>(() =>
    calculateRemainingTime(targetIso, offsetMs)
  );

  useEffect(() => {
    setRemaining(calculateRemainingTime(targetIso, offsetMs));

    if (!targetIso) return;

    const interval = setInterval(() => {
      const next = calculateRemainingTime(targetIso, offsetMs);
      setRemaining(next);
      if (next.isExpired) {
        clearInterval(interval);
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [targetIso, offsetMs]);

  return remaining;
}
