import { useState } from 'react';
import { Coins, ShieldAlert, Sparkles, LogOut, CheckCircle2 } from 'lucide-react';
import type { User } from '~/api/types';
import { formatCoins } from '~/lib/money';

interface BalanceCardProps {
  user: User;
  onClaimDaily: () => Promise<unknown>;
  onLogout: () => void;
}

export function BalanceCard({ user, onClaimDaily, onLogout }: BalanceCardProps) {
  const [claiming, setClaiming] = useState(false);
  const [claimed, setClaimed] = useState(false);

  const handleClaim = async () => {
    setClaiming(true);
    try {
      await onClaimDaily();
      setClaimed(true);
      setTimeout(() => setClaimed(false), 3000);
    } finally {
      setClaiming(false);
    }
  };

  const available = user.balance - user.hold;

  return (
    <div className="bg-white rounded-2xl border border-slate-200/80 p-5 shadow-xs space-y-4">
      {/* Пользователь и аватар */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3.5">
          <div className="w-12 h-12 rounded-2xl bg-teal-600 text-white font-black text-lg flex items-center justify-center shadow-xs">
            {user.name.charAt(0).toUpperCase()}
          </div>
          <div>
            <h2 className="text-base sm:text-lg font-bold text-slate-900">{user.name}</h2>
            <p className="text-xs text-slate-400">{user.email}</p>
          </div>
        </div>

        <button
          onClick={onLogout}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs text-slate-500 hover:text-rose-600 hover:bg-rose-50 rounded-xl transition-colors font-medium"
        >
          <LogOut className="w-3.5 h-3.5" />
          <span>Выйти</span>
        </button>
      </div>

      {/* Блок балансов */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 pt-3 border-t border-slate-100">
        {/* Баланс Coins */}
        <div className="group relative p-3 rounded-xl bg-amber-50/70 border border-amber-200/70 cursor-help">
          <div className="flex items-center gap-1 text-[11px] font-semibold text-amber-800">
            <Coins className="w-3.5 h-3.5 text-amber-600" />
            <span>Баланс монет</span>
          </div>
          <div className="text-lg font-black text-amber-950 mt-1">
            {formatCoins(user.balance)}
          </div>
          <div className="absolute left-0 bottom-full mb-1 hidden group-hover:block p-2 bg-slate-900 text-white text-[11px] rounded-lg shadow-lg z-50 pointer-events-none w-52">
            Пополняется автоматически при ежедневном заходе на сайт.
          </div>
        </div>

        {/* Доступно */}
        <div className="p-3 rounded-xl bg-teal-50/70 border border-teal-200/70">
          <div className="text-[11px] font-semibold text-teal-800">Доступно для ставок</div>
          <div className="text-lg font-black text-teal-950 mt-1">
            {formatCoins(available)}
          </div>
        </div>

        {/* Заморожено (Hold) */}
        <div className="group relative p-3 rounded-xl bg-slate-100/80 border border-slate-200/80 cursor-help">
          <div className="flex items-center gap-1 text-[11px] font-semibold text-slate-700">
            <ShieldAlert className="w-3.5 h-3.5 text-slate-500" />
            <span>Заморожено (Hold)</span>
          </div>
          <div className="text-lg font-black text-slate-800 mt-1">
            {formatCoins(user.hold)}
          </div>
          <div className="absolute left-0 bottom-full mb-1 hidden group-hover:block p-2 bg-slate-900 text-white text-[11px] rounded-lg shadow-lg z-50 pointer-events-none w-60">
            Замороженные средства для активных ставок. Размораживаются после перебивания или завершения.
          </div>
        </div>
      </div>

      {/* Ежедневный бонус */}
      <div className="pt-2 flex items-center justify-between">
        <div className="flex items-center gap-1.5 text-xs text-slate-500">
          <Sparkles className="w-4 h-4 text-amber-500" />
          <span>Ежедневная награда: <strong>+1 000 🪙</strong></span>
        </div>

        <button
          onClick={handleClaim}
          disabled={claiming}
          className="flex items-center gap-1.5 px-3 py-1.5 bg-amber-500 hover:bg-amber-600 active:scale-98 text-white text-xs font-bold rounded-xl shadow-xs transition-all disabled:opacity-50"
        >
          {claimed ? (
            <>
              <CheckCircle2 className="w-3.5 h-3.5" />
              <span>Получено!</span>
            </>
          ) : (
            <>
              <Coins className="w-3.5 h-3.5" />
              <span>{claiming ? 'Начисление...' : 'Забрать +1 000 🪙'}</span>
            </>
          )}
        </button>
      </div>
    </div>
  );
}
