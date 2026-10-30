import React from 'react';
import { ArrowDownLeft, ArrowUpRight, History, RefreshCw } from 'lucide-react';
import type { LedgerEntry } from '~/api/types';
import { formatCoins } from '~/lib/money';
import { formatDate } from '~/lib/date';

interface TransactionsTableProps {
  transactions: LedgerEntry[];
  onRefresh?: () => void;
  refreshing?: boolean;
}

export function TransactionsTable({ transactions, onRefresh, refreshing }: TransactionsTableProps) {
  return (
    <div className="bg-white rounded-2xl border border-slate-200/80 p-5 shadow-xs space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 font-bold text-slate-900 text-sm sm:text-base">
          <History className="w-4 h-4 text-slate-500" />
          <span>История операций</span>
          <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-slate-100 text-slate-600">
            {transactions.length}
          </span>
        </div>

        {onRefresh && (
          <button
            type="button"
            onClick={onRefresh}
            disabled={refreshing}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs text-slate-600 hover:text-teal-700 bg-slate-50 hover:bg-teal-50 border border-slate-200 rounded-xl transition-colors font-medium cursor-pointer disabled:opacity-50"
            title="Обновить историю операций с сервера"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${refreshing ? 'animate-spin text-teal-600' : ''}`} />
            <span>{refreshing ? 'Загрузка...' : 'Обновить'}</span>
          </button>
        )}
      </div>

      {transactions.length === 0 ? (
        <p className="text-xs text-slate-400 text-center py-4">Операций пока не совершалось.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="text-[11px] font-bold text-slate-400 uppercase tracking-wider border-b border-slate-100 pb-2">
              <tr>
                <th className="py-2.5 px-3">Тип операции</th>
                <th className="py-2.5 px-3">Дата</th>
                <th className="py-2.5 px-3 text-right">Сумма</th>
                <th className="py-2.5 px-3 text-right">Остаток</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {transactions.map((tx) => {
                const isPositive = tx.amount > 0;
                let title = 'Операция';
                if (tx.type === 'top_up') title = 'Ежедневное начисление';
                else if (tx.type === 'payout_of_winnings') title = 'Выплата за победу в лоте';
                else if (tx.type === 'transfer_to_seller') title = 'Списание за покупку лота';

                return (
                  <tr key={tx.id} className="hover:bg-slate-50 transition-colors">
                    <td className="py-2.5 px-3 flex items-center gap-2 font-medium text-slate-800">
                      <div
                        className={`w-6 h-6 rounded-lg flex items-center justify-center shrink-0 ${
                          isPositive
                            ? 'bg-emerald-50 text-emerald-600'
                            : 'bg-rose-50 text-rose-600'
                        }`}
                      >
                        {isPositive ? (
                          <ArrowDownLeft className="w-3.5 h-3.5" />
                        ) : (
                          <ArrowUpRight className="w-3.5 h-3.5" />
                        )}
                      </div>
                      <span>{title}</span>
                    </td>
                    <td className="py-2.5 px-3 text-slate-400 whitespace-nowrap">
                      {formatDate(tx.created_at)}
                    </td>
                    <td
                      className={`py-2.5 px-3 text-right font-bold whitespace-nowrap ${
                        isPositive ? 'text-emerald-600' : 'text-slate-800'
                      }`}
                    >
                      {isPositive ? '+' : ''}
                      {formatCoins(tx.amount)}
                    </td>
                    <td className="py-2.5 px-3 text-right font-semibold text-slate-500 whitespace-nowrap">
                      {formatCoins(tx.balance_after)}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
