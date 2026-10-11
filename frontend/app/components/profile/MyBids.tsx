import React from 'react';
import { Award } from 'lucide-react';
import type { Lot } from '~/api/types';
import { LotCard } from '~/components/lot/LotCard';

interface MyBidsProps {
  lots: Lot[];
}

export function MyBids({ lots }: MyBidsProps) {
  return (
    <div className="bg-white rounded-2xl border border-slate-200/80 p-5 shadow-xs space-y-4">
      <div className="flex items-center gap-2 font-bold text-slate-900 text-sm sm:text-base">
        <Award className="w-4 h-4 text-amber-500" />
        <span>Мои ставки (Участие в торгах)</span>
        <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-slate-100 text-slate-600">
          {lots.length}
        </span>
      </div>

      {lots.length === 0 ? (
        <p className="text-xs text-slate-400 text-center py-4">
          Вы пока не делали ставок в активных лотах.
        </p>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-2 md:grid-cols-3 gap-2.5 sm:gap-4">
          {lots.map((lot) => (
            <LotCard key={lot.id} lot={lot} highlightLeader={true} />
          ))}
        </div>
      )}
    </div>
  );
}
