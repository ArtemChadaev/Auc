import { useState } from 'react';
import { Link, useNavigate } from 'react-router';
import {
  Gavel,
  PlusCircle,
  Edit3,
  Play,
  CheckCircle2,
  AlertCircle,
  Loader2,
  Clock,
  Eye,
  Sparkles,
  XCircle,
} from 'lucide-react';
import type { Lot, LotStatus } from '~/api/types';
import { formatCoins } from '~/lib/money';
import { activateLot } from '~/api/lots';
import { formatTime } from '~/lib/date';

interface MyLotsProps {
  lots: Lot[];
  onRefresh?: () => void;
}

type TabKey = 'all' | 'draft' | 'pending_review' | 'approved' | 'active' | 'finished' | 'rejected';

export function MyLots({ lots, onRefresh }: MyLotsProps) {
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState<TabKey>('all');
  const [activatingId, setActivatingId] = useState<number | null>(null);
  const [actionSuccess, setActionSuccess] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  // Неначавшиеся / недоделанные лоты
  const unstartedLots = lots.filter(
    (l) =>
      l.status === 'draft' ||
      l.status === 'pending_review' ||
      l.status === 'approved' ||
      l.status === 'rejected'
  );

  const draftLots = lots.filter((l) => l.status === 'draft');
  const pendingLots = lots.filter((l) => l.status === 'pending_review');
  const approvedLots = lots.filter((l) => l.status === 'approved');
  const activeLots = lots.filter((l) => l.status === 'active');
  const finishedLots = lots.filter((l) => l.status === 'finished');
  const rejectedLots = lots.filter((l) => l.status === 'rejected');

  const filteredLots = lots.filter((l) => {
    if (activeTab === 'all') return true;
    return l.status === activeTab;
  });

  const handleActivate = async (lotId: number) => {
    setActivatingId(lotId);
    setActionSuccess(null);
    setActionError(null);
    try {
      await activateLot(lotId);
      setActionSuccess(`Лот #${lotId} успешно активирован! Торги начались.`);
      setTimeout(() => setActionSuccess(null), 4000);
      onRefresh?.();
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : 'Не удалось активировать лот');
      setTimeout(() => setActionError(null), 5000);
    } finally {
      setActivatingId(null);
    }
  };

  const getStatusBadge = (status: LotStatus) => {
    switch (status) {
      case 'draft':
        return (
          <span className="px-2 py-0.5 text-[10px] font-bold rounded-md bg-slate-100 text-slate-700 border border-slate-300">
            Черновик
          </span>
        );
      case 'pending_review':
        return (
          <span className="px-2 py-0.5 text-[10px] font-bold rounded-md bg-amber-50 text-amber-700 border border-amber-300">
            На модерации
          </span>
        );
      case 'approved':
        return (
          <span className="px-2 py-0.5 text-[10px] font-bold rounded-md bg-sky-50 text-sky-700 border border-sky-300 flex items-center gap-1">
            <Sparkles className="w-2.5 h-2.5 text-sky-600" />
            <span>Одобрен (готов к запуску)</span>
          </span>
        );
      case 'active':
        return (
          <span className="px-2 py-0.5 text-[10px] font-bold rounded-md bg-emerald-50 text-emerald-700 border border-emerald-300">
            Идут торги (Active)
          </span>
        );
      case 'rejected':
        return (
          <span className="px-2 py-0.5 text-[10px] font-bold rounded-md bg-rose-50 text-rose-700 border border-rose-300 flex items-center gap-1">
            <XCircle className="w-2.5 h-2.5 text-rose-600" />
            <span>Отклонён модерацией</span>
          </span>
        );
      case 'finished':
        return (
          <span className="px-2 py-0.5 text-[10px] font-bold rounded-md bg-slate-100 text-slate-600 border border-slate-200">
            Завершён
          </span>
        );
      case 'cancelled':
        return (
          <span className="px-2 py-0.5 text-[10px] font-bold rounded-md bg-rose-50 text-rose-700 border border-rose-200">
            Отменён
          </span>
        );
      default:
        return (
          <span className="px-2 py-0.5 text-[10px] font-bold rounded-md bg-slate-100 text-slate-600">
            {status}
          </span>
        );
    }
  };

  return (
    <div className="bg-white rounded-2xl border border-slate-200/80 p-5 shadow-xs space-y-4">
      {/* Заголовок и быстрые действия */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div className="flex items-center gap-2 font-bold text-slate-900 text-sm sm:text-base">
          <Gavel className="w-4 h-4 text-teal-600" />
          <span>Мои лоты</span>
          <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-slate-100 text-slate-600">
            {lots.length}
          </span>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {/* Быстрый выбор неначавшихся лотов для редактирования */}
          {unstartedLots.length > 0 && (
            <select
              value=""
              onChange={(e) => {
                if (e.target.value) {
                  navigate(`/lots/new?edit_id=${e.target.value}`);
                }
              }}
              className="text-xs bg-slate-50 hover:bg-slate-100 border border-slate-200 text-slate-700 rounded-xl px-2.5 py-1.5 focus:outline-hidden focus:ring-1 focus:ring-teal-500 font-medium"
            >
              <option value="" disabled>
                Изменить неначавшийся лот ({unstartedLots.length})...
              </option>
              {unstartedLots.map((l) => (
                <option key={l.id} value={l.id}>
                  #{l.id}: {l.title} ({l.status})
                </option>
              ))}
            </select>
          )}

          <Link
            to="/lots/new"
            className="flex items-center gap-1.5 px-3 py-1.5 bg-teal-600 hover:bg-teal-700 text-white text-xs font-bold rounded-xl shadow-xs transition-colors"
          >
            <PlusCircle className="w-3.5 h-3.5" />
            <span>Выставить лот</span>
          </Link>
        </div>
      </div>

      {/* Уведомления об успехе / ошибке */}
      {actionSuccess && (
        <div className="p-3 bg-emerald-50 border border-emerald-200 rounded-xl text-emerald-900 text-xs font-semibold flex items-center gap-2 animate-in fade-in">
          <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
          <span>{actionSuccess}</span>
        </div>
      )}
      {actionError && (
        <div className="p-3 bg-rose-50 border border-rose-200 rounded-xl text-rose-900 text-xs font-semibold flex items-center gap-2 animate-in fade-in">
          <AlertCircle className="w-4 h-4 text-rose-600 shrink-0" />
          <span>{actionError}</span>
        </div>
      )}

      {/* Вкладки / фильтры статусов */}
      <div className="flex items-center gap-1.5 overflow-x-auto pb-1 text-xs border-b border-slate-100">
        <button
          type="button"
          onClick={() => setActiveTab('all')}
          className={`px-3 py-1.5 rounded-lg font-medium transition-colors cursor-pointer shrink-0 ${
            activeTab === 'all'
              ? 'bg-slate-900 text-white font-bold'
              : 'text-slate-600 hover:bg-slate-100'
          }`}
        >
          Все ({lots.length})
        </button>

        {draftLots.length > 0 && (
          <button
            type="button"
            onClick={() => setActiveTab('draft')}
            className={`px-3 py-1.5 rounded-lg font-medium transition-colors cursor-pointer shrink-0 ${
              activeTab === 'draft'
                ? 'bg-slate-700 text-white font-bold'
                : 'text-slate-600 hover:bg-slate-100'
            }`}
          >
            Черновики ({draftLots.length})
          </button>
        )}

        {pendingLots.length > 0 && (
          <button
            type="button"
            onClick={() => setActiveTab('pending_review')}
            className={`px-3 py-1.5 rounded-lg font-medium transition-colors cursor-pointer shrink-0 ${
              activeTab === 'pending_review'
                ? 'bg-amber-600 text-white font-bold'
                : 'text-slate-600 hover:bg-slate-100'
            }`}
          >
            На модерации ({pendingLots.length})
          </button>
        )}

        {approvedLots.length > 0 && (
          <button
            type="button"
            onClick={() => setActiveTab('approved')}
            className={`px-3 py-1.5 rounded-lg font-medium transition-colors cursor-pointer shrink-0 ${
              activeTab === 'approved'
                ? 'bg-sky-600 text-white font-bold'
                : 'text-sky-700 hover:bg-sky-50'
            }`}
          >
            Одобрены к запуску ({approvedLots.length})
          </button>
        )}

        {activeLots.length > 0 && (
          <button
            type="button"
            onClick={() => setActiveTab('active')}
            className={`px-3 py-1.5 rounded-lg font-medium transition-colors cursor-pointer shrink-0 ${
              activeTab === 'active'
                ? 'bg-emerald-600 text-white font-bold'
                : 'text-slate-600 hover:bg-slate-100'
            }`}
          >
            Активные ({activeLots.length})
          </button>
        )}

        {rejectedLots.length > 0 && (
          <button
            type="button"
            onClick={() => setActiveTab('rejected')}
            className={`px-3 py-1.5 rounded-lg font-medium transition-colors cursor-pointer shrink-0 ${
              activeTab === 'rejected'
                ? 'bg-rose-600 text-white font-bold'
                : 'text-rose-700 hover:bg-rose-50'
            }`}
          >
            Отклонённые ({rejectedLots.length})
          </button>
        )}

        {finishedLots.length > 0 && (
          <button
            type="button"
            onClick={() => setActiveTab('finished')}
            className={`px-3 py-1.5 rounded-lg font-medium transition-colors cursor-pointer shrink-0 ${
              activeTab === 'finished'
                ? 'bg-slate-800 text-white font-bold'
                : 'text-slate-600 hover:bg-slate-100'
            }`}
          >
            Завершённые ({finishedLots.length})
          </button>
        )}
      </div>

      {/* Список лотов */}
      {filteredLots.length === 0 ? (
        <div className="py-8 text-center text-xs text-slate-400">
          В выбранной категории лоты не найдены.
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3.5">
          {filteredLots.map((lot) => {
            const isApproved = lot.status === 'approved';
            const isRejected = lot.status === 'rejected';
            const isEditable = lot.status === 'draft' || lot.status === 'pending_review' || isRejected;
            const isActivating = activatingId === lot.id;

            return (
              <div
                key={lot.id}
                className={`p-3.5 rounded-2xl border transition-all flex flex-col justify-between gap-3 ${
                  isApproved
                    ? 'bg-sky-50/40 border-sky-200 shadow-xs'
                    : isRejected
                    ? 'bg-rose-50/30 border-rose-200 shadow-xs'
                    : 'bg-white border-slate-200/80 hover:border-slate-300'
                }`}
              >
                <div>
                  <div className="flex items-center justify-between gap-2 mb-2">
                    {getStatusBadge(lot.status)}
                    <span className="text-[11px] font-mono text-slate-400">#{lot.id}</span>
                  </div>

                  <h4 className="text-sm font-bold text-slate-900 line-clamp-1">{lot.title}</h4>

                  {lot.description?.short_text && (
                    <p className="text-xs text-slate-500 line-clamp-2 mt-0.5">
                      {lot.description.short_text}
                    </p>
                  )}

                  {isRejected && lot.description?.reject_reason && (
                    <div className="p-2 bg-rose-50 border border-rose-200 rounded-lg text-[11px] text-rose-700 mt-2 flex items-start gap-1.5">
                      <AlertCircle className="w-3.5 h-3.5 text-rose-600 shrink-0 mt-0.5" />
                      <span>Причина отклонения: {lot.description.reject_reason}</span>
                    </div>
                  )}

                  <div className="grid grid-cols-2 gap-2 mt-3 pt-2.5 border-t border-slate-100 text-xs">
                    <div>
                      <span className="text-[11px] text-slate-400 block">Стартовая цена:</span>
                      <span className="font-bold text-slate-800">{formatCoins(lot.start_amount)}</span>
                    </div>

                    <div>
                      <span className="text-[11px] text-slate-400 block">
                        {lot.status === 'active' ? 'Текущая ставка:' : 'Мин. шаг:'}
                      </span>
                      <span className="font-bold text-teal-700">
                        {lot.status === 'active'
                          ? formatCoins(lot.current_bid)
                          : formatCoins(lot.min_step)}
                      </span>
                    </div>
                  </div>

                  {lot.started_at && (
                    <div className="text-[11px] text-slate-400 mt-2 flex items-center gap-1">
                      <Clock className="w-3 h-3 text-slate-400" />
                      <span>Старт: {formatTime(lot.started_at)}</span>
                    </div>
                  )}
                </div>

                {/* Блок действий с лотом */}
                <div className="pt-2 border-t border-slate-100 flex items-center justify-end gap-2">
                  {/* Кнопка активации approved лота */}
                  {isApproved && (
                    <button
                      type="button"
                      onClick={() => handleActivate(lot.id)}
                      disabled={isActivating}
                      className="flex-1 flex items-center justify-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs rounded-xl shadow-xs transition-colors cursor-pointer disabled:opacity-50"
                      title="Перевести лот в статус active и начать торги"
                    >
                      {isActivating ? (
                        <Loader2 className="w-3.5 h-3.5 animate-spin" />
                      ) : (
                        <Play className="w-3.5 h-3.5 fill-current" />
                      )}
                      <span>Активировать торги (Active)</span>
                    </button>
                  )}

                  {/* Кнопка редактирования черновика или лота на модерации / отклонённого */}
                  {isEditable && (
                    <Link
                      to={`/lots/new?edit_id=${lot.id}`}
                      className="flex items-center gap-1.5 px-3 py-1.5 bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold text-xs rounded-xl transition-colors"
                    >
                      <Edit3 className="w-3.5 h-3.5 text-slate-500" />
                      <span>Редактировать</span>
                    </Link>
                  )}

                  <Link
                    to={`/lot/${lot.id}`}
                    className="flex items-center gap-1 px-3 py-1.5 text-xs text-teal-700 hover:bg-teal-50 border border-teal-200 rounded-xl transition-colors font-medium"
                  >
                    <Eye className="w-3.5 h-3.5" />
                    <span>Просмотр</span>
                  </Link>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
