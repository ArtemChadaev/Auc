import { useState, useEffect, useCallback } from 'react';
import { useParams, Link } from 'react-router';
import {
  ArrowLeft,
  Clock,
  ShieldCheck,
  Sparkles,
  AlertCircle,
  Play,
  Edit3,
  Loader2,
  CheckCircle2,
} from 'lucide-react';
import { getLot, activateLot } from '~/api/lots';
import { listBids } from '~/api/bids';
import type { Lot, Bid } from '~/api/types';
import { useLotStream } from '~/realtime/useLotStream';
import { useAuth } from '~/hooks/useAuth';
import { Gallery } from '~/components/lot/Gallery';
import { BidForm } from '~/components/lot/BidForm';
import { BidHistory } from '~/components/lot/BidHistory';
import { LotInfo } from '~/components/lot/LotInfo';
import { CountdownBadge } from '~/components/lot/CountdownBadge';
import { ErrorAlert } from '~/components/feedback/ErrorAlert';
import { EmptyState } from '~/components/feedback/EmptyState';

export default function LotDetailPage() {
  const { lotId } = useParams();
  const idNumber = Number(lotId);

  const { user } = useAuth();

  const [lot, setLot] = useState<Lot | null>(null);
  const [bids, setBids] = useState<Bid[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown | null>(null);
  const [antiSnipingExtended, setAntiSnipingExtended] = useState(false);

  // Активация лота продавцом
  const [activating, setActivating] = useState(false);
  const [activateSuccess, setActivateSuccess] = useState<string | null>(null);
  const [activateError, setActivateError] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    if (isNaN(idNumber)) return;
    setLoading(true);
    setError(null);
    try {
      const [lotData, bidsData] = await Promise.all([getLot(idNumber), listBids(idNumber)]);
      setLot(lotData);
      setBids(bidsData);
    } catch (err) {
      setError(err);
    } finally {
      setLoading(false);
    }
  }, [idNumber]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleActivateLot = async () => {
    if (!lot) return;
    setActivating(true);
    setActivateError(null);
    setActivateSuccess(null);
    try {
      await activateLot(lot.id);
      setActivateSuccess('Лот успешно активирован! Торги открыты.');
      await loadData();
    } catch (err: unknown) {
      setActivateError(err instanceof Error ? err.message : 'Не удалось активировать лот');
    } finally {
      setActivating(false);
    }
  };

  // Слушаем realtime-события конкретного лота (новые ставки, Anti-sniping продление, финиш)
  useLotStream(idNumber || null, (event) => {
    if (event.type === 'bid_placed' && event.data.bid) {
      setBids((prev) => [event.data.bid!, ...prev]);
      setLot((prev) => {
        if (!prev) return null;
        return {
          ...prev,
          current_bid: event.data.current_bid ?? prev.current_bid,
          bids_count: event.data.bids_count ?? prev.bids_count,
          ends_at: event.data.ends_at ?? prev.ends_at,
        };
      });
    } else if (event.type === 'time_extended') {
      if (event.data.bid) {
        setBids((prev) => [event.data.bid!, ...prev]);
      }
      setLot((prev) => {
        if (!prev) return null;
        return {
          ...prev,
          current_bid: event.data.current_bid ?? prev.current_bid,
          bids_count: event.data.bids_count ?? prev.bids_count,
          ends_at: event.data.ends_at ?? prev.ends_at,
        };
      });
      // Показываем индикатор Anti-sniping продления
      setAntiSnipingExtended(true);
      setTimeout(() => setAntiSnipingExtended(false), 8000);
    } else if (event.type === 'lot_finished') {
      setLot((prev) => {
        if (!prev) return null;
        return {
          ...prev,
          status: 'finished',
          winner: event.data.winner ?? prev.winner,
          finish_amount: event.data.finish_amount ?? prev.finish_amount,
          finished_at: event.data.finished_at ?? prev.finished_at,
        };
      });
    }
  });

  if (loading) {
    return (
      <div className="py-12 flex flex-col items-center justify-center space-y-3">
        <div className="w-8 h-8 border-3 border-teal-600 border-t-transparent rounded-full animate-spin" />
        <p className="text-xs text-slate-500 font-medium">Загрузка информации о лоте...</p>
      </div>
    );
  }

  if (error || !lot) {
    return (
      <div className="py-8 space-y-4">
        <Link
          to="/lots"
          className="inline-flex items-center gap-1.5 text-xs font-semibold text-slate-600 hover:text-slate-900 transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          <span>Назад в каталог</span>
        </Link>
        <ErrorAlert error={error || 'Лот не найден'} />
        <EmptyState
          title="Лот не найден"
          description="Возможно, он был перемещен или удален"
          action={
            <Link
              to="/lots"
              className="px-4 py-2 bg-teal-600 text-white rounded-xl text-xs font-semibold shadow-xs"
            >
              В каталог
            </Link>
          }
        />
      </div>
    );
  }

  const isSeller = Boolean(user?.id && user.id === lot.seller_id);

  return (
    <div className="space-y-4 sm:space-y-6 pb-12">
      {/* Навигационная цепочка / назад */}
      <div className="flex items-center justify-between">
        <Link
          to="/lots"
          className="inline-flex items-center gap-1.5 text-xs font-semibold text-slate-600 hover:text-teal-700 transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          <span>Каталог лотов</span>
        </Link>

        <div className="flex items-center gap-2">
          <CountdownBadge endsAt={lot.ends_at} status={lot.status} />
        </div>
      </div>

      {/* Панель продавца: Активация лота в статусе approved */}
      {isSeller && lot.status === 'approved' && (
        <div className="p-4 bg-sky-50 border border-sky-200 rounded-2xl flex flex-col sm:flex-row sm:items-center justify-between gap-3 shadow-xs animate-in fade-in">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-sky-600 text-white flex items-center justify-center font-bold shrink-0">
              <Sparkles className="w-5 h-5" />
            </div>
            <div>
              <h4 className="font-bold text-sm text-sky-950">Лот одобрен модератором!</h4>
              <p className="text-xs text-sky-800">
                Вы можете перевести лот в статус <strong>active</strong> прямо сейчас, чтобы открыть прием ставок.
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={handleActivateLot}
            disabled={activating}
            className="flex items-center justify-center gap-1.5 px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs rounded-xl shadow-xs transition-colors cursor-pointer disabled:opacity-50 shrink-0"
          >
            {activating ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Play className="w-4 h-4 fill-current" />
            )}
            <span>Активировать торги (Сделать active)</span>
          </button>
        </div>
      )}

      {/* Панель продавца: Редактирование черновика или лота на модерации */}
      {isSeller && (lot.status === 'draft' || lot.status === 'pending_review') && (
        <div className="p-4 bg-amber-50 border border-amber-200 rounded-2xl flex flex-col sm:flex-row sm:items-center justify-between gap-3 shadow-xs animate-in fade-in">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-amber-500 text-white flex items-center justify-center font-bold shrink-0">
              <Clock className="w-5 h-5" />
            </div>
            <div>
              <h4 className="font-bold text-sm text-amber-950">
                {lot.status === 'draft' ? 'Черновик лота' : 'Лот находится на модерации'}
              </h4>
              <p className="text-xs text-amber-800">
                Торги еще не начались. Вы можете изменить название, описание, цены и даты.
              </p>
            </div>
          </div>
          <Link
            to={`/lots/new?edit_id=${lot.id}`}
            className="flex items-center justify-center gap-1.5 px-4 py-2 bg-amber-600 hover:bg-amber-700 text-white font-bold text-xs rounded-xl shadow-xs transition-colors shrink-0"
          >
            <Edit3 className="w-4 h-4" />
            <span>Редактировать лот</span>
          </Link>
        </div>
      )}

      {activateSuccess && (
        <div className="p-3 bg-emerald-50 border border-emerald-200 rounded-xl text-emerald-900 text-xs font-semibold flex items-center gap-2">
          <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
          <span>{activateSuccess}</span>
        </div>
      )}
      {activateError && (
        <div className="p-3 bg-rose-50 border border-rose-200 rounded-xl text-rose-900 text-xs font-semibold flex items-center gap-2">
          <AlertCircle className="w-4 h-4 text-rose-600 shrink-0" />
          <span>{activateError}</span>
        </div>
      )}

      {/* Баннер Anti-sniping при продлении */}
      {antiSnipingExtended && (
        <div className="p-3.5 bg-amber-50 border border-amber-300 rounded-xl text-amber-900 text-xs font-semibold flex items-center justify-between shadow-xs animate-bounce">
          <div className="flex items-center gap-2">
            <Sparkles className="w-4 h-4 text-amber-600 shrink-0" />
            <span>Время аукциона продлено сервером (Anti-sniping защита)!</span>
          </div>
          <span className="text-[11px] font-normal text-amber-700 hidden sm:inline">
            Ставка на последних секундах продлевает торги
          </span>
        </div>
      )}

      {/* Победитель лота (если завершён) */}
      {lot.status === 'finished' && lot.winner && (
        <div className="p-4 bg-emerald-50 border border-emerald-200 rounded-2xl text-emerald-950 flex items-center justify-between shadow-xs">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-emerald-600 text-white flex items-center justify-center font-black text-base shadow-xs">
              ✓
            </div>
            <div>
              <h4 className="font-bold text-sm">Аукцион завершён!</h4>
              <p className="text-xs text-emerald-800">
                Победитель: <strong>{lot.winner.name}</strong>
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Основная сетка: Галерея + Форма ставок */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-5 sm:gap-6 items-start">
        {/* Левая колонка */}
        <div className="lg:col-span-7 space-y-4">
          <Gallery description={lot.description} title={lot.title} category={lot.category} />

          {/* На мобильном: форма ставки сразу под фото! */}
          <div className="lg:hidden">
            <BidForm lot={lot} onBidSuccess={loadData} />
          </div>

          <LotInfo lot={lot} />
        </div>

        {/* Правая колонка: на десктопе sticky форма + история ставок на обоих */}
        <div className="lg:col-span-5 space-y-4 lg:sticky lg:top-20">
          <div className="hidden lg:block">
            <BidForm lot={lot} onBidSuccess={loadData} />
          </div>
          <BidHistory
            bids={bids}
            currentUserId={user?.id}
            startAmount={lot.start_amount}
            startedAt={lot.started_at || lot.created_at}
          />
        </div>
      </div>
    </div>
  );
}
