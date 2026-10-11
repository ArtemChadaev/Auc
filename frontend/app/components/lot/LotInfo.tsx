import type { Lot } from '~/api/types';
import { formatCoins } from '~/lib/money';
import { formatDate } from '~/lib/date';

interface LotInfoProps {
  lot: Lot;
}

export function LotInfo({ lot }: LotInfoProps) {
  return (
    <div className="bg-white rounded-xl border border-slate-200/80 p-4 sm:p-5 shadow-xs space-y-4">
      {/* Продавец */}
      <div className="flex items-center justify-between pb-3 border-b border-slate-100">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-lg bg-slate-200 flex items-center justify-center font-bold text-slate-700 text-xs">
            {lot.seller.name.charAt(0).toUpperCase()}
          </div>
          <div>
            <span className="font-semibold text-slate-900 text-xs sm:text-sm">{lot.seller.name}</span>
            <p className="text-[10px] text-slate-400">Продавец</p>
          </div>
        </div>

        <div className="text-right">
          <span className="text-[10px] text-slate-400">Начало</span>
          <p className="text-[11px] font-medium text-slate-600">
            {formatDate(lot.started_at || lot.created_at)}
          </p>
        </div>
      </div>

      {/* Описание */}
      <div className="space-y-1.5">
        <h4 className="text-[11px] font-bold uppercase tracking-wider text-slate-500">
          Описание цифрового лота
        </h4>
        <p className="text-xs sm:text-sm text-slate-700 leading-relaxed whitespace-pre-line">
          {lot.description.text}
        </p>
      </div>

      {/* Характеристики (Атрибуты файла) */}
      {lot.description.attributes && Object.keys(lot.description.attributes).length > 0 && (
        <div className="space-y-2 pt-2 border-t border-slate-100">
          <h4 className="text-[11px] font-bold uppercase tracking-wider text-slate-500">
            Свойства файла
          </h4>
          <dl className="grid grid-cols-1 sm:grid-cols-2 gap-1.5 text-xs">
            {Object.entries(lot.description.attributes).map(([key, val]) => (
              <div key={key} className="flex justify-between py-1 px-2.5 rounded-lg bg-slate-50">
                <dt className="text-slate-500">{key}</dt>
                <dd className="font-semibold text-slate-800 truncate ml-2">{val}</dd>
              </div>
            ))}
          </dl>
        </div>
      )}

      {/* Стартовые условия */}
      <div className="pt-2 border-t border-slate-100 flex items-center justify-between text-[11px] text-slate-400">
        <div>
          Стартовая цена: <strong className="text-slate-700">{formatCoins(lot.start_amount)}</strong>
        </div>
        <div>Хранилище: S3 Asset #{lot.item_id}</div>
      </div>
    </div>
  );
}
