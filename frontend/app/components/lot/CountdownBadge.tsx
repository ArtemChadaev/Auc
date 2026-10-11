import React from 'react';
import { Clock, AlertCircle } from 'lucide-react';
import { useCountdown } from '~/hooks/useCountdown';

interface CountdownBadgeProps {
  endsAt: string | null | undefined;
  startedAt?: string | null | undefined;
  status: string;
  className?: string;
}

export function CountdownBadge({ endsAt, startedAt, status, className = '' }: CountdownBadgeProps) {
  const isUpcoming = Boolean(
    startedAt &&
    new Date(startedAt).getTime() > Date.now() &&
    status !== 'finished' &&
    status !== 'cancelled'
  );

  const countdown = useCountdown(isUpcoming ? startedAt : endsAt);

  if (status === 'finished') {
    return (
      <span className={`inline-flex items-center gap-1 px-1.5 sm:px-2.5 py-0.5 sm:py-1 rounded-full text-[10px] sm:text-xs font-semibold bg-slate-100 text-slate-600 ${className}`}>
        Завершён
      </span>
    );
  }

  if (status === 'cancelled') {
    return (
      <span className={`inline-flex items-center gap-1 px-1.5 sm:px-2.5 py-0.5 sm:py-1 rounded-full text-[10px] sm:text-xs font-semibold bg-rose-50 text-rose-600 ${className}`}>
        Отменён
      </span>
    );
  }

  // Неначавшийся лот (желтый цвет)
  if (isUpcoming && !countdown.isExpired) {
    return (
      <span
        className={`inline-flex items-center gap-1 px-1.5 sm:px-2 py-0.5 sm:py-0.5 rounded-full text-[10px] sm:text-xs font-semibold bg-amber-50 text-amber-700 border border-amber-300 shadow-2xs ${className}`}
      >
        <Clock className="w-2.5 sm:w-3.5 h-2.5 sm:h-3.5 text-amber-600" />
        <span>До старта: {countdown.formatted}</span>
      </span>
    );
  }

  if (status === 'draft' || !endsAt) {
    return (
      <span className={`inline-flex items-center gap-1 px-1.5 sm:px-2.5 py-0.5 sm:py-1 rounded-full text-[10px] sm:text-xs font-semibold bg-slate-100 text-slate-500 ${className}`}>
        Черновик
      </span>
    );
  }

  if (countdown.isExpired) {
    return (
      <span className={`inline-flex items-center gap-1 px-1.5 sm:px-2.5 py-0.5 sm:py-1 rounded-full text-[10px] sm:text-xs font-semibold bg-amber-50 text-amber-700 animate-pulse ${className}`}>
        Итоги...
      </span>
    );
  }

  const isUrgent = countdown.totalSeconds < 120; // менее 2 минут

  return (
    <span
      className={`inline-flex items-center gap-1 px-1.5 sm:px-2 py-0.5 sm:py-0.5 rounded-full text-[10px] sm:text-xs font-semibold transition-colors ${
        isUrgent
          ? 'bg-rose-50 text-rose-600 border border-rose-200/80 animate-pulse'
          : 'bg-teal-50 text-teal-800 border border-teal-200/60'
      } ${className}`}
    >
      {isUrgent ? <AlertCircle className="w-2.5 sm:w-3.5 h-2.5 sm:h-3.5" /> : <Clock className="w-2.5 sm:w-3.5 h-2.5 sm:h-3.5 text-teal-600" />}
      <span>{countdown.formatted}</span>
    </span>
  );
}
