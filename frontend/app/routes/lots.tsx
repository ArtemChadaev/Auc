import { useState, useEffect, useCallback } from 'react';
import { useSearchParams } from 'react-router';
import { ArrowUp, Loader2, ChevronDown, CheckCircle2 } from 'lucide-react';
import { listLots } from '~/api/lots';
import type { Lot, LotFilter } from '~/api/types';
import { usePublicStream } from '~/realtime/usePublicStream';
import { LotFilters } from '~/components/lot/LotFilters';
import { LotGrid } from '~/components/lot/LotGrid';
import { LotGridSkeleton } from '~/components/feedback/LotSkeleton';
import { EmptyState } from '~/components/feedback/EmptyState';
import { ErrorAlert } from '~/components/feedback/ErrorAlert';

const INITIAL_FILTER: LotFilter = {
  status: 'active', // По умолчанию только активные
  category: '',
  query: '',
  sort: 'newest',
  page: 1,
  limit: 6,
};

export default function LotsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const initialStatus = searchParams.get('status') || 'active';

  const [filter, setFilter] = useState<LotFilter>(() => ({
    ...INITIAL_FILTER,
    status: initialStatus,
  }));
  const [lots, setLots] = useState<Lot[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<unknown | null>(null);

  // Загрузка первой страницы при изменении фильтров
  const fetchLots = useCallback(async (currentFilter: LotFilter) => {
    setLoading(true);
    setError(null);
    try {
      const res = await listLots({ ...currentFilter, page: 1 });
      setLots(res.items);
      setTotal(res.total);
      setPage(1);
    } catch (err) {
      setError(err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchLots(filter);
  }, [filter, fetchLots]);

  // Подгрузка следующей страницы ("Показать ещё")
  const loadMore = async () => {
    if (loadingMore || lots.length >= total) return;
    setLoadingMore(true);
    try {
      const nextPage = page + 1;
      const res = await listLots({ ...filter, page: nextPage });
      setLots((prev) => [...prev, ...res.items]);
      setPage(nextPage);
      setTotal(res.total);
    } catch (err) {
      setError(err);
    } finally {
      setLoadingMore(false);
    }
  };

  // Реалтайм обновления из публичного потока
  usePublicStream((event) => {
    if (event.type === 'lot_updated') {
      setLots((prev) =>
        prev.map((lot) => {
          if (lot.id === event.lot_id) {
            return {
              ...lot,
              current_bid: event.data.current_bid ?? lot.current_bid,
              bids_count: event.data.bids_count ?? lot.bids_count,
              ends_at: event.data.ends_at ?? lot.ends_at,
            };
          }
          return lot;
        })
      );
    } else if (event.type === 'lot_finished') {
      setLots((prev) =>
        prev.map((lot) => {
          if (lot.id === event.lot_id) {
            return {
              ...lot,
              status: 'finished',
              winner: event.data.winner ?? lot.winner,
              finish_amount: event.data.finish_amount ?? lot.finish_amount,
            };
          }
          return lot;
        })
      );
    }
  });

  const scrollToTop = () => {
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const hasMore = lots.length < total;

  return (
    <div className="space-y-5 pb-12">
      {/* Заголовок страницы */}
      <div className="flex flex-col gap-1">
        <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-slate-900">
          Каталог лотов
        </h1>
      </div>

      {/* Панель фильтров */}
      <LotFilters
        filter={filter}
        onChange={(newFilter) => setFilter(newFilter)}
        onReset={() => setFilter(INITIAL_FILTER)}
      />

      {/* Ошибка */}
      {error ? <ErrorAlert error={error} onDismiss={() => setError(null)} /> : null}

      {/* Сетка лотов или лоадер */}
      {loading ? (
        <LotGridSkeleton count={6} />
      ) : lots.length === 0 ? (
        <EmptyState
          title="Лоты не найдены"
          description="Попробуйте сбросить фильтры или выбрать другой статус"
          action={
            <button
              onClick={() => setFilter(INITIAL_FILTER)}
              className="px-4 py-2 bg-teal-600 hover:bg-teal-700 text-white text-xs font-semibold rounded-xl shadow-xs transition-colors"
            >
              Сбросить фильтры
            </button>
          }
        />
      ) : (
        <div className="space-y-6">
          <LotGrid lots={lots} />

          {/* Пагинация: Индикатор и кнопка «Показать ещё» */}
          <div className="flex flex-col items-center justify-center pt-4 pb-2 gap-3">
            {hasMore ? (
              <button
                type="button"
                onClick={loadMore}
                disabled={loadingMore}
                className="flex items-center justify-center gap-2 px-6 py-2.5 bg-white hover:bg-slate-50 active:bg-slate-100 text-slate-800 font-semibold text-xs sm:text-sm border border-slate-200 hover:border-slate-300 rounded-xl shadow-xs hover:shadow-sm transition-all disabled:opacity-50"
              >
                {loadingMore ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin text-teal-600" />
                    <span>Загрузка...</span>
                  </>
                ) : (
                  <span>Показать ещё</span>
                )}
              </button>
            ) : null}
          </div>
        </div>
      )}

      {/* Кнопка «Вернуться в начало» (стрелка вверх), появляется при второй и последующих дозагрузках (page >= 2) */}
      {page >= 2 && (
        <button
          type="button"
          onClick={scrollToTop}
          title="Вернуться в начало"
          className="fixed bottom-20 right-4 sm:right-8 md:bottom-8 z-30 p-2.5 rounded-full bg-teal-600 text-white shadow-lg hover:bg-teal-700 hover:scale-105 active:scale-95 transition-all"
        >
          <ArrowUp className="w-5 h-5" />
        </button>
      )}
    </div>
  );
}
