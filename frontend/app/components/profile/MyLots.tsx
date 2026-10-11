import { Link } from 'react-router';
import { Gavel, PlusCircle } from 'lucide-react';
import type { Lot } from '~/api/types';
import { LotCard } from '~/components/lot/LotCard';

interface MyLotsProps {
  lots: Lot[];
}

export function MyLots({ lots }: MyLotsProps) {
  return (
    <div className="bg-white rounded-2xl border border-slate-200/80 p-5 shadow-xs space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 font-bold text-slate-900 text-sm sm:text-base">
          <Gavel className="w-4 h-4 text-teal-600" />
          <span>Мои лоты</span>
          <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-slate-100 text-slate-600">
            {lots.length}
          </span>
        </div>

        <Link
          to="/lots/new"
          className="flex items-center gap-1.5 px-3 py-1.5 bg-teal-600 hover:bg-teal-700 text-white text-xs font-bold rounded-xl shadow-xs transition-colors"
        >
          <PlusCircle className="w-3.5 h-3.5" />
          <span>Выставить лот</span>
        </Link>
      </div>

      {lots.length === 0 ? (
        <p className="text-xs text-slate-400 text-center py-4">
          Вы пока не выставили ни одного лота на продажу.
        </p>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-2 md:grid-cols-3 gap-2.5 sm:gap-4">
          {lots.map((lot) => (
            <LotCard key={lot.id} lot={lot} />
          ))}
        </div>
      )}
    </div>
  );
}
