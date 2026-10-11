import React from 'react';
import { History, Award, Coins } from 'lucide-react';
import type { Bid } from '~/api/types';
import { formatCoins } from '~/lib/money';
import { formatTime } from '~/lib/date';

interface BidHistoryProps {
  bids: Bid[];
  currentUserId?: string;
  startAmount?: number;
  startedAt?: string;
}

export function BidHistory({ bids, currentUserId, startAmount, startedAt }: BidHistoryProps) {
  return (
    <div className="bg-white rounded-2xl border border-slate-200/80 p-5 shadow-xs space-y-3.5">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 font-semibold text-slate-800 text-sm">
          <History className="w-4 h-4 text-slate-500" />
          <span>История ставок</span>
        </div>
        <span className="text-xs text-slate-400 font-medium">
          {bids.length > 0 ? `Всего: ${bids.length}` : 'Торги открыты'}
        </span>
      </div>

      <div className="divide-y divide-slate-100 max-h-72 overflow-y-auto pr-1">
        {/* Ставки участников (от новых к старым) */}
        {bids.map((bid, index) => {
          const isHighest = index === 0;
          const isMine = currentUserId && bid.user_id === currentUserId;

          return (
            <div
              key={bid.id}
              className={`py-2.5 flex items-center justify-between text-xs transition-colors ${
                isHighest ? 'bg-amber-50/50 -mx-2 px-2 rounded-lg' : ''
              }`}
            >
              <div className="flex items-center gap-2 min-w-0">
                <div
                  className={`w-6 h-6 rounded-full flex items-center justify-center font-bold text-[10px] shrink-0 ${
                    isHighest
                      ? 'bg-amber-500 text-white'
                      : isMine
                      ? 'bg-teal-600 text-white'
                      : 'bg-slate-200 text-slate-700'
                  }`}
                >
                  {bid.user_name ? bid.user_name.charAt(0).toUpperCase() : '?'}
                </div>

                <div className="flex flex-col min-w-0">
                  <div className="flex items-center gap-1.5 truncate">
                    <span className="font-semibold text-slate-800 truncate">
                      {bid.user_name || 'Участник'}
                    </span>
                    {isHighest && (
                      <span className="inline-flex items-center gap-0.5 px-1.5 py-0.2 rounded-md bg-amber-100 text-amber-800 text-[10px] font-bold shrink-0">
                        <Award className="w-3 h-3 text-amber-600" />
                        Лидер
                      </span>
                    )}
                    {isMine && !isHighest && (
                      <span className="px-1.5 py-0.2 rounded-md bg-teal-100 text-teal-800 text-[10px] font-semibold shrink-0">
                        Вы
                      </span>
                    )}
                  </div>
                  <span className="text-[10px] text-slate-400">{formatTime(bid.created_at)}</span>
                </div>
              </div>

              <div className="flex items-center gap-1 font-bold text-slate-900 shrink-0 ml-2">
                <Coins className="w-3.5 h-3.5 text-amber-500" />
                <span>{formatCoins(bid.amount)}</span>
              </div>
            </div>
          );
        })}

        {/* Начальная цена в самом начале истории */}
        {startAmount !== undefined && (
          <div className="py-2.5 flex items-center justify-between text-xs text-slate-500 bg-slate-50/60 -mx-1 px-2 rounded-lg mt-1">
            <div className="flex items-center gap-2 min-w-0">
              <div className="w-6 h-6 rounded-full bg-slate-100 border border-slate-200 flex items-center justify-center font-bold text-[10px] text-slate-500 shrink-0">
                ★
              </div>
              <div className="flex flex-col min-w-0">
                <span className="font-semibold text-slate-600">Начальная цена лота</span>
                {startedAt && (
                  <span className="text-[10px] text-slate-400">{formatTime(startedAt)}</span>
                )}
              </div>
            </div>

            <div className="flex items-center gap-1 font-bold text-slate-700 shrink-0 ml-2">
              <Coins className="w-3.5 h-3.5 text-slate-400" />
              <span>{formatCoins(startAmount)}</span>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
