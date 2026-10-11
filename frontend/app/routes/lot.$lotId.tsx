import { useState, useEffect, useCallback } from 'react';
import { useParams, Link } from 'react-router';
import { ArrowLeft, Clock, ShieldCheck, Sparkles, AlertCircle } from 'lucide-react';
import { getLot } from '~/api/lots';
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
