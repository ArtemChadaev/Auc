import { useState, useEffect } from 'react';
import { Link } from 'react-router';
import { Coins, Loader2, Plus, Sparkles, AlertCircle } from 'lucide-react';
import type { Lot } from '~/api/types';
import { placeBid } from '~/api/bids';
import { activateLot, cancelLot } from '~/api/lots';
import { useAuth } from '~/hooks/useAuth';
import { formatCoins } from '~/lib/money';
import { ErrorAlert } from '~/components/feedback/ErrorAlert';
import { recordParticipatingLot } from '~/lib/participating-lots';

interface BidFormProps {
  lot: Lot;
  onBidSuccess: () => void;
}

export function BidForm({ lot, onBidSuccess }: BidFormProps) {
  const { user, isAuthenticated } = useAuth();
  const minRequiredBid = lot.current_bid + lot.min_step;

  const [amount, setAmount] = useState<number>(minRequiredBid);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<unknown | null>(null);
  const [successNotice, setSuccessNotice] = useState<string | null>(null);
  const [sellerActionLoading, setSellerActionLoading] = useState(false);
  const [sellerActionError, setSellerActionError] = useState<string | null>(null);

  const handleActivate = async () => {
    setSellerActionLoading(true);
    setSellerActionError(null);
    try {
      await activateLot(lot.id);
      setSuccessNotice('Лот успешно активирован!');
      onBidSuccess();
    } catch (err: unknown) {
      setSellerActionError(err instanceof Error ? err.message : 'Не удалось активировать лот');
    } finally {
      setSellerActionLoading(false);
    }
  };

  const handleCancel = async () => {
    if (!confirm('Вы уверены, что хотите отменить этот лот?')) return;
    setSellerActionLoading(true);
    setSellerActionError(null);
    try {
      await cancelLot(lot.id);
      setSuccessNotice('Лот успешно отменен!');
      onBidSuccess();
    } catch (err: unknown) {
      setSellerActionError(err instanceof Error ? err.message : 'Не удалось отменить лот');
    } finally {
      setSellerActionLoading(false);
    }
  };

  // Обновляем сумму ставки, если текущая ставка выросла
  useEffect(() => {
    if (amount < minRequiredBid) {
      setAmount(minRequiredBid);
    }
  }, [minRequiredBid, amount]);

  const availableBalance = user ? user.balance - user.hold : 0;
  const isSeller = user && user.id === lot.seller_id;
  const isFinished = lot.status === 'finished' || lot.status === 'cancelled';

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (submitting || !isAuthenticated || isSeller || isFinished) return;

    setError(null);
    setSuccessNotice(null);
    setSubmitting(true);

    try {
      await placeBid(lot.id, amount);
      if (user?.id) {
        recordParticipatingLot(lot.id, user.id);
      }
      setSuccessNotice(`Ставка ${formatCoins(amount)} успешно принята!`);
      onBidSuccess();
      setTimeout(() => setSuccessNotice(null), 3000);
    } catch (err) {
      setError(err);
      // Если опередили или ставка мала, подставляем актуальный минимум
      setAmount(lot.current_bid + lot.min_step);
    } finally {
      setSubmitting(false);
    }
  };

  const addStep = (multiplier: number) => {
    const next = lot.current_bid + lot.min_step * multiplier;
    setAmount(next);
  };

  return (
    <div className="bg-white rounded-2xl border border-slate-200/80 p-5 shadow-xs space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <span className="text-xs text-slate-500 font-medium uppercase tracking-wider">
            {isFinished ? 'Итоговая ставка' : 'Текущая цена'}
          </span>
          <div className="flex items-center gap-1.5 text-2xl font-black text-slate-900 mt-0.5">
            <Coins className="w-6 h-6 text-amber-500" />
            <span>{formatCoins(lot.current_bid)}</span>
          </div>
        </div>

        <div className="text-right">
          <span className="text-xs text-slate-500 font-medium">Минимальный шаг</span>
          <div className="text-sm font-semibold text-slate-700 mt-0.5">
            +{formatCoins(lot.min_step)}
          </div>
        </div>
      </div>

      {/* Ошибки или успешные уведомления */}
      {error ? <ErrorAlert error={error} onDismiss={() => setError(null)} /> : null}
      {successNotice && (
        <div className="p-3 bg-emerald-50 border border-emerald-200 rounded-xl text-emerald-900 text-xs font-semibold flex items-center gap-2">
          <Sparkles className="w-4 h-4 text-emerald-600 shrink-0" />
          <span>{successNotice}</span>
        </div>
      )}

      {/* Завершённый лот */}
      {isFinished ? (
        <div className="p-4 bg-slate-50 rounded-xl text-center border border-slate-200/60">
          <span className="text-sm font-semibold text-slate-700">Торги по лоту закрыты</span>
          {lot.winner && (
            <p className="text-xs text-slate-500 mt-1">
              Победитель: <strong className="text-slate-800">{lot.winner.name}</strong>
            </p>
          )}
        </div>
      ) : !isAuthenticated ? (
        /* Неавторизованный пользователь */
        <div className="p-4 bg-slate-50 rounded-xl text-center border border-slate-200/60 space-y-2">
          <p className="text-xs text-slate-600 font-medium">
            Войдите в аккаунт, чтобы делать ставки в монетах Coins
          </p>
          <Link
            to="/login"
            className="inline-flex items-center justify-center px-4 py-2 bg-teal-600 hover:bg-teal-700 text-white text-xs font-semibold rounded-xl shadow-xs transition-colors"
          >
            Войти в аккаунт
          </Link>
        </div>
      ) : isSeller ? (
        /* Продавец своего лота */
        <div className="p-4 bg-amber-50 border border-amber-200/80 rounded-xl text-amber-900 text-xs font-medium flex items-center gap-2">
          <AlertCircle className="w-4 h-4 text-amber-600 shrink-0" />
          <span>Вы являетесь продавцом этого лота и не можете делать ставки на него.</span>
        </div>
      ) : (
        /* Форма ставки */
        <form onSubmit={handleSubmit} noValidate className="space-y-3.5">
          {/* Индикатор доступного баланса */}
          <div className="flex items-center justify-between text-xs px-1">
            <span className="text-slate-500">Доступно для ставки:</span>
            <span
              className={`font-semibold ${
                availableBalance < amount ? 'text-rose-600' : 'text-slate-800'
              }`}
            >
              {formatCoins(availableBalance)}
            </span>
          </div>

          {/* Быстрые кнопки увеличения ставки (+шаг, +2×шаг, +5×шаг) */}
          <div className="grid grid-cols-3 gap-2">
            <button
              type="button"
              onClick={() => addStep(1)}
              className="py-1.5 px-2 bg-slate-100 hover:bg-slate-200 text-slate-700 text-xs font-semibold rounded-lg transition-colors flex items-center justify-center gap-1"
            >
              <Plus className="w-3 h-3 text-slate-500" />
              <span>1 шаг</span>
            </button>
            <button
              type="button"
              onClick={() => addStep(2)}
              className="py-1.5 px-2 bg-slate-100 hover:bg-slate-200 text-slate-700 text-xs font-semibold rounded-lg transition-colors flex items-center justify-center gap-1"
            >
              <Plus className="w-3 h-3 text-slate-500" />
              <span>2 шага</span>
            </button>
            <button
              type="button"
              onClick={() => addStep(5)}
              className="py-1.5 px-2 bg-slate-100 hover:bg-slate-200 text-slate-700 text-xs font-semibold rounded-lg transition-colors flex items-center justify-center gap-1"
            >
              <Plus className="w-3 h-3 text-slate-500" />
              <span>5 шагов</span>
            </button>
          </div>

          {/* Поле ввода суммы */}
          <div>
            <div className="relative">
              <input
                type="number"
                min={minRequiredBid}
                step={lot.min_step}
                value={amount}
                onChange={(e) => setAmount(Number(e.target.value))}
                className={`w-full pl-3.5 pr-14 py-2.5 text-base font-bold bg-slate-50 border rounded-xl focus:outline-hidden focus:ring-2 transition-all text-slate-900 ${
                  amount < minRequiredBid || availableBalance < amount
                    ? 'border-rose-400 bg-rose-50/20 focus:ring-rose-400/20'
                    : 'border-slate-200 focus:ring-teal-500/20 focus:border-teal-500'
                }`}
              />
              <span className="absolute right-3.5 top-1/2 -translate-y-1/2 text-xs font-bold text-slate-400 pointer-events-none">
                🪙 Coins
              </span>
            </div>
            {amount < minRequiredBid ? (
              <p className="mt-1 text-xs text-rose-600 font-medium">
                Минимальная ставка: {formatCoins(minRequiredBid)} Coins
              </p>
            ) : availableBalance < amount ? (
              <p className="mt-1 text-xs text-rose-600 font-medium">
                Недостаточно средств. Доступно: {formatCoins(availableBalance)} Coins
              </p>
            ) : null}
          </div>

          {/* Кнопка отправки ставки */}
          <button
            type="submit"
            disabled={submitting || amount < minRequiredBid}
            className="w-full py-3 bg-teal-600 hover:bg-teal-700 active:scale-99 text-white font-bold text-sm rounded-xl shadow-md transition-all flex items-center justify-center gap-2 disabled:opacity-50 disabled:pointer-events-none"
          >
            {submitting ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                <span>Отправка ставки...</span>
              </>
            ) : (
              <>
                <Coins className="w-4 h-4 text-amber-300" />
                <span>Сделать ставку {formatCoins(amount)}</span>
              </>
            )}
          </button>
        </form>
      )}
    </div>
  );
}
