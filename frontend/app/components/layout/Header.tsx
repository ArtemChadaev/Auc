import { useState, useRef, useEffect } from 'react';
import { Link, useLocation } from 'react-router';
import {
  Gavel,
  PlusCircle,
  User as UserIcon,
  Coins,
  ShieldAlert,
  Sparkles,
  ChevronDown,
  Info,
} from 'lucide-react';
import { useAuth } from '~/hooks/useAuth';
import { formatCoins } from '~/lib/money';
import { usePublicStream } from '~/realtime/usePublicStream';

export function Header() {
  const { user, isAuthenticated, claimDailyCoins } = useAuth();
  const { status } = usePublicStream();
  const location = useLocation();

  const [balancePopoverOpen, setBalancePopoverOpen] = useState(false);
  const popoverRef = useRef<HTMLDivElement>(null);

  const isLive = status.state === 'connected';

  // Закрытие поповера при клике вне
  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (popoverRef.current && !popoverRef.current.contains(e.target as Node)) {
        setBalancePopoverOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const available = user ? user.balance - user.hold : 0;

  return (
    <header className="sticky top-0 z-40 w-full bg-white/95 backdrop-blur-md border-b border-slate-200/80 transition-all">
      <div className="max-w-5xl mx-auto px-4 h-14 sm:h-15 flex items-center justify-between gap-2.5">
        {/* Логотип */}
        <Link to="/lots" className="flex items-center gap-2 font-bold text-slate-900 group shrink-0">
          <div className="w-8.5 h-8.5 rounded-xl bg-teal-600 text-white flex items-center justify-center shadow-xs group-hover:bg-teal-700 transition-colors">
            <Gavel className="w-4.5 h-4.5" />
          </div>
          <div className="flex flex-col">
            <span className="text-sm sm:text-base tracking-tight leading-none font-bold">Аукцион</span>
            <span className="text-[10px] text-slate-500 font-normal leading-tight mt-0.5">Файлы & Ассеты</span>
          </div>
        </Link>

        {/* Навигация для десктопа */}
        <nav className="hidden md:flex items-center gap-1 text-xs sm:text-sm font-medium">
          <Link
            to="/lots"
            className={`px-3 py-1.5 rounded-lg transition-colors ${
              location.pathname.startsWith('/lots') && location.pathname !== '/lots/new'
                ? 'bg-slate-100 text-slate-900 font-semibold'
                : 'text-slate-600 hover:text-slate-900 hover:bg-slate-50'
            }`}
          >
            Каталог файлов
          </Link>
          <Link
            to="/lots/new"
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-colors ${
              location.pathname === '/lots/new'
                ? 'bg-slate-100 text-slate-900 font-semibold'
                : 'text-slate-600 hover:text-slate-900 hover:bg-slate-50'
            }`}
          >
            <PlusCircle className="w-4 h-4 text-teal-600" />
            Выставить лот
          </Link>
        </nav>

        {/* Правая часть: балансы, live статус и профиль */}
        <div className="flex items-center gap-2 sm:gap-2.5">
          {/* Индикатор SSE Live */}
          <div
            title={isLive ? 'SSE поток активен' : 'Переподключение...'}
            className="flex items-center gap-1 px-2 py-1 rounded-full bg-slate-100 text-[11px] font-medium text-slate-600"
          >
            <span
              className={`w-2 h-2 rounded-full ${
                isLive ? 'bg-emerald-500 animate-pulse' : 'bg-amber-500'
              }`}
            />
            <span className="hidden sm:inline">{isLive ? 'Live' : 'Подкл.'}</span>
          </div>

          {/* Интерактивный счетчик Coins и Hold */}
          {isAuthenticated && user && (
            <div className="relative" ref={popoverRef}>
              <button
                type="button"
                onClick={() => setBalancePopoverOpen((prev) => !prev)}
                className="flex items-center gap-1.5 px-2.5 py-1 rounded-xl bg-amber-50/90 hover:bg-amber-100/90 border border-amber-200 text-amber-950 text-xs font-bold transition-all shadow-2xs cursor-pointer"
                title="Нажмите для подробностей баланса и Hold"
              >
                <Coins className="w-3.5 h-3.5 text-amber-600 shrink-0" />
                <span>{formatCoins(user.balance)}</span>

                {/* Отображение Hold прямо в плашке */}
                {user.hold > 0 && (
                  <span className="flex items-center gap-0.5 pl-1 ml-1 border-l border-amber-300 text-[11px] text-amber-800 font-semibold">
                    <ShieldAlert className="w-3 h-3 text-amber-700" />
                    <span>{formatCoins(user.hold)}</span>
                  </span>
                )}

                <ChevronDown className={`w-3 h-3 text-amber-700 transition-transform ${balancePopoverOpen ? 'rotate-180' : ''}`} />
              </button>

              {/* Выпадающее окно детализации баланса и Hold */}
              {balancePopoverOpen && (
                <div className="absolute right-0 top-full mt-2 w-72 p-4 bg-white rounded-2xl border border-slate-200 shadow-xl z-50 text-xs space-y-3 animate-in fade-in zoom-in-95 duration-100">
                  <div className="flex items-center justify-between pb-2 border-b border-slate-100">
                    <span className="font-bold text-slate-900 text-sm">Финансы Coins</span>
                    <button
                      onClick={() => claimDailyCoins()}
                      className="flex items-center gap-1 px-2 py-0.5 rounded-lg bg-teal-50 text-teal-800 font-bold hover:bg-teal-100 transition-colors text-[11px]"
                      title="Забрать ежедневные монеты"
                    >
                      <Sparkles className="w-3 h-3 text-amber-500" />
                      <span>+1 000 🪙</span>
                    </button>
                  </div>

                  <div className="space-y-2">
                    <div className="flex justify-between items-center py-1">
                      <span className="text-slate-500 font-medium">Общий баланс:</span>
                      <span className="font-bold text-slate-900">{formatCoins(user.balance)}</span>
                    </div>

                    <div className="flex justify-between items-center py-1 px-2 rounded-lg bg-emerald-50 text-emerald-950 font-semibold">
                      <span>Доступно для ставок:</span>
                      <span className="font-bold text-emerald-700">{formatCoins(available)}</span>
                    </div>

                    <div className="flex justify-between items-center py-1 px-2 rounded-lg bg-amber-50 text-amber-950 font-semibold">
                      <span className="flex items-center gap-1">
                        <ShieldAlert className="w-3.5 h-3.5 text-amber-600" />
                        <span>Заморожено (Hold):</span>
                      </span>
                      <span className="font-bold text-amber-700">{formatCoins(user.hold)}</span>
                    </div>
                  </div>

                  <div className="p-2.5 rounded-xl bg-slate-50 text-[11px] text-slate-500 leading-relaxed border border-slate-100 flex items-start gap-1.5">
                    <Info className="w-3.5 h-3.5 text-slate-400 shrink-0 mt-0.5" />
                    <span>
                      Сумма ставки удерживается (Hold) пока вы лидируете. Если ставку перебивают, средства мгновенно возвращаются на доступный баланс.
                    </span>
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Профиль или Вход */}
          {isAuthenticated && user ? (
            <Link
              to="/profile"
              className="flex items-center gap-1.5 pl-1 pr-2 py-1 rounded-xl hover:bg-slate-100 transition-colors"
            >
              <div className="w-7 h-7 rounded-full bg-teal-700 text-white text-xs font-bold flex items-center justify-center">
                {user.name.charAt(0).toUpperCase()}
              </div>
              <span className="hidden md:inline text-xs font-medium text-slate-700 max-w-28 truncate">
                {user.name}
              </span>
            </Link>
          ) : (
            <Link
              to="/login"
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-teal-600 hover:bg-teal-700 text-white text-xs font-semibold shadow-xs transition-colors"
            >
              <UserIcon className="w-3.5 h-3.5" />
              <span>Войти</span>
            </Link>
          )}
        </div>
      </div>
    </header>
  );
}
