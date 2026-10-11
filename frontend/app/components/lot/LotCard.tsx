import { useState, useEffect } from 'react';
import { Link } from 'react-router';
import { Users, Coins, Sparkles, FileBox, FileCode } from 'lucide-react';
import type { Lot } from '~/api/types';
import { formatCoins } from '~/lib/money';
import { CountdownBadge } from './CountdownBadge';

interface LotCardProps {
  lot: Lot;
  highlightLeader?: boolean;
  interactive?: boolean;
}

export function LotCard({ lot, highlightLeader, interactive = true }: LotCardProps) {
  const [flashing, setFlashing] = useState(false);
  const [prevPrice, setPrevPrice] = useState(lot.current_bid);

  useEffect(() => {
    if (lot.current_bid !== prevPrice) {
      setPrevPrice(lot.current_bid);
      setFlashing(true);
      const timer = setTimeout(() => setFlashing(false), 1200);
      return () => clearTimeout(timer);
    }
  }, [lot.current_bid, prevPrice]);

  const cardContent = (
    <>
      {/* Компактное превью с соотношением 16:9 */}
      <div className="relative aspect-16/10 w-full bg-slate-100 overflow-hidden">
        <img
          src="/placeholder.svg"
          alt={lot.title}
          className={`w-full h-full object-cover transition-transform duration-300 ${
            interactive ? 'group-hover:scale-103' : ''
          }`}
          loading="lazy"
        />

        {/* Бейдж типа файла и категории */}
        <div className="absolute top-1.5 left-1.5 flex items-center gap-1">
          <span className="px-1.5 py-0.5 rounded-md bg-white/95 backdrop-blur-xs text-[9px] sm:text-[10px] font-bold text-slate-700 shadow-xs flex items-center gap-1 max-w-20 sm:max-w-none truncate">
            <FileBox className="w-2.5 sm:w-3 h-2.5 sm:h-3 text-teal-600 shrink-0" />
            <span className="truncate">{lot.category}</span>
          </span>
        </div>

        {/* Таймер (отображается только если интерактивная карточка в каталоге) */}
        {interactive && (
          <div className="absolute top-1.5 right-1.5">
            <CountdownBadge endsAt={lot.ends_at} startedAt={lot.started_at} status={lot.status} />
          </div>
        )}

        {highlightLeader && (
          <div className="absolute bottom-2 left-2 flex items-center gap-1 px-2 py-0.5 rounded-md bg-emerald-600 text-white text-[10px] font-bold shadow-xs">
            <Sparkles className="w-2.5 h-2.5" />
            Вы лидируете
          </div>
        )}
      </div>

      {/* Компактная информация о лоте */}
      <div className="p-3 flex flex-col flex-1 justify-between gap-2.5">
        <div>
          <h3
            className={`font-semibold text-slate-900 line-clamp-1 text-xs sm:text-sm transition-colors ${
              interactive ? 'group-hover:text-teal-700' : ''
            }`}
          >
            {lot.title}
          </h3>
          {lot.description.short_text && (
            <p className="mt-0.5 text-[11px] text-slate-500 line-clamp-1 leading-snug">
              {lot.description.short_text}
            </p>
          )}
        </div>

        <div className="pt-2 border-t border-slate-100 flex items-end justify-between">
          <div className="flex flex-col">
            <span className="text-[9px] text-slate-400 uppercase tracking-wider font-semibold">
              {lot.status === 'finished' ? 'Итоговая ставка' : 'Текущая ставка'}
            </span>
            <div className="flex items-center gap-1 font-bold text-slate-900 text-xs sm:text-sm">
              <Coins className="w-3.5 h-3.5 text-amber-500" />
              <span className={flashing ? 'text-teal-600 transition-colors' : ''}>
                {formatCoins(lot.current_bid)}
              </span>
            </div>
            {lot.min_step ? (
              <span className="text-[10px] text-slate-400 font-medium mt-0.5">
                Шаг: +{formatCoins(lot.min_step)}
              </span>
            ) : null}
          </div>

          <div className="flex items-center gap-1 text-[11px] text-slate-400 font-medium pb-0.5">
            <Users className="w-3 h-3" />
            <span>{lot.bids_count}</span>
          </div>
        </div>
      </div>
    </>
  );

  const containerClasses = `group flex flex-col bg-white rounded-xl border transition-all duration-200 overflow-hidden shadow-xs ${
    interactive ? 'hover:shadow-md hover:border-teal-500/50 cursor-pointer' : 'cursor-default'
  } ${
    flashing ? 'ring-2 ring-teal-500 border-teal-500 bg-teal-50/20' : 'border-slate-200/80'
  }`;

  if (!interactive) {
    return <div className={containerClasses}>{cardContent}</div>;
  }

  return (
    <Link to={`/lot/${lot.id}`} className={containerClasses}>
      {cardContent}
    </Link>
  );
}
