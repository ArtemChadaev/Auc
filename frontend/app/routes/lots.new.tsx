import { useState, useEffect } from 'react';
import { useNavigate, useSearchParams, Link } from 'react-router';
import {
  Gavel,
  Loader2,
  Plus,
  Eye,
  ArrowLeft,
  Calendar,
  Clock,
  AlertCircle,
  Sparkles,
  Edit3,
  Play,
  RotateCcw,
  CheckCircle2,
} from 'lucide-react';
import { useAuth } from '~/hooks/useAuth';
import { listUserItems } from '~/api/items';
import { createLot, getMyLots, updateLot, activateLot, getLot, cacheLot } from '~/api/lots';
import type { Item, Lot } from '~/api/types';
import { LotCard } from '~/components/lot/LotCard';
import { ErrorAlert } from '~/components/feedback/ErrorAlert';

// Соответствие типа предмета категории аукциона
export const ITEM_TYPE_TO_CATEGORY: Record<string, string> = {
  '3d': '3D Модели',
  image: 'Цифровой Арт',
  audio: 'Аудио',
  video: 'Видео',
  document: 'Документы',
  archive: 'Архивы',
};

type QuickDuration = 'hour' | 'day' | 'week';

// Вспомогательная функция для форматирования в строку вида YYYY-MM-DDTHH:mm
function toLocalISOString(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0');
  const y = date.getFullYear();
  const m = pad(date.getMonth() + 1);
  const d = pad(date.getDate());
  const hh = pad(date.getHours());
  const mm = pad(date.getMinutes());
  return `${y}-${m}-${d}T${hh}:${mm}`;
}

// По умолчанию: следующий день, 08:00 утра
function getDefaultStartDate(): Date {
  const d = new Date();
  d.setDate(d.getDate() + 1);
  d.setHours(8, 0, 0, 0);
  if (d.getTime() - Date.now() < 3600 * 1000) {
    d.setTime(Date.now() + 3600 * 1000 + 10 * 60 * 1000);
  }
  return d;
}

function calculateEndDate(start: Date, duration: QuickDuration): Date {
  const end = new Date(start.getTime());
  if (duration === 'hour') {
    end.setHours(end.getHours() + 1);
  } else if (duration === 'day') {
    end.setDate(end.getDate() + 1);
  } else if (duration === 'week') {
    end.setDate(end.getDate() + 7);
  }
  return end;
}

export default function CreateLotPage() {
  const { user, isAuthenticated, loading: authLoading } = useAuth();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const preselectedItemId = searchParams.get('item_id');
  const editIdParam = searchParams.get('edit_id');

  const [items, setItems] = useState<Item[]>([]);
  const [myLots, setMyLots] = useState<Lot[]>([]);

  // Режим редактирования существующего лота
  const [editingLotId, setEditingLotId] = useState<number | null>(
    editIdParam ? Number(editIdParam) : null
  );
  const [editingLotStatus, setEditingLotStatus] = useState<string | null>(null);

  const [selectedItemId, setSelectedItemId] = useState<number | null>(
    preselectedItemId ? Number(preselectedItemId) : null
  );

  const selectedItem = items.find((it) => it.id === selectedItemId);
  const category = selectedItem
    ? ITEM_TYPE_TO_CATEGORY[selectedItem.type] || 'Цифровой Арт'
    : 'Цифровой Арт';

  const [title, setTitle] = useState('');
  const [shortText, setShortText] = useState('');
  const [text, setText] = useState('');
  const [startAmount, setStartAmount] = useState(5000);
  const [minStep, setMinStep] = useState(250);

  // Галочки draft и auto_activate
  const [isDraft, setIsDraft] = useState(false);
  const [autoActivate, setAutoActivate] = useState(true);

  // Даты старта и окончания
  const [startDateStr, setStartDateStr] = useState(() => toLocalISOString(getDefaultStartDate()));
  const [quickDuration, setQuickDuration] = useState<QuickDuration>('day');
  const [endDateStr, setEndDateStr] = useState(() =>
    toLocalISOString(calculateEndDate(getDefaultStartDate(), 'day'))
  );

  const [submitting, setSubmitting] = useState(false);
  const [activating, setActivating] = useState(false);
  const [activateSuccess, setActivateSuccess] = useState<string | null>(null);
  const [error, setError] = useState<unknown | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  // Загрузка предметов и лотов пользователя
  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      navigate('/login');
      return;
    }
    if (isAuthenticated) {
      Promise.all([listUserItems().catch(() => []), getMyLots().catch(() => [])])
        .then(([itemsData, lotsData]) => {
          setItems(itemsData);
          setMyLots(lotsData);

          // Если передан edit_id через URL, загружаем его данные
          if (editIdParam) {
            const numEditId = Number(editIdParam);
            const foundLot = lotsData.find((l) => l.id === numEditId);
            if (foundLot) {
              loadLotIntoForm(foundLot);
            } else {
              getLot(numEditId)
                .then(loadLotIntoForm)
                .catch(() => {});
            }
          } else if (!selectedItemId && itemsData.length > 0) {
            setSelectedItemId(itemsData[0].id);
            if (itemsData[0].metadata?.title) {
              setTitle(itemsData[0].metadata.title as string);
            }
          }
        })
        .catch(() => {});
    }
  }, [authLoading, isAuthenticated, navigate, editIdParam]);

  // Неначавшиеся или недоделанные лоты (черновики, на модерации, одобренные)
  const unstartedLots = myLots.filter(
    (l) => l.status === 'draft' || l.status === 'pending_review' || l.status === 'approved'
  );

  const loadLotIntoForm = (lot: Lot) => {
    setEditingLotId(lot.id);
    setEditingLotStatus(lot.status);
    setSelectedItemId(lot.item_id);
    setTitle(lot.title || '');
    setShortText(lot.description?.short_text || '');
    setText(lot.description?.text || '');
    setStartAmount(lot.start_amount || 5000);
    setMinStep(lot.min_step || 250);
    setIsDraft(lot.status === 'draft');
    setAutoActivate(lot.description?.auto_activate ?? true);
    if (lot.started_at) {
      setStartDateStr(toLocalISOString(new Date(lot.started_at)));
    }
    if (lot.ends_at) {
      setEndDateStr(toLocalISOString(new Date(lot.ends_at)));
    }
  };

  const resetToCreateNew = () => {
    setEditingLotId(null);
    setEditingLotStatus(null);
    setSearchParams({});
    setTitle('');
    setShortText('');
    setText('');
    setStartAmount(5000);
    setMinStep(250);
    setIsDraft(false);
    setAutoActivate(true);
    setStartDateStr(toLocalISOString(getDefaultStartDate()));
    setEndDateStr(toLocalISOString(calculateEndDate(getDefaultStartDate(), 'day')));
    if (items.length > 0) {
      setSelectedItemId(items[0].id);
      if (items[0].metadata?.title) {
        setTitle(items[0].metadata.title as string);
      }
    }
  };

  // При смене даты старта пересчитываем дату конца согласно выбранному радио
  const handleStartDateChange = (newStartStr: string) => {
    setStartDateStr(newStartStr);
    if (fieldErrors.startDate) {
      setFieldErrors((prev) => ({ ...prev, startDate: '' }));
    }
    const startObj = new Date(newStartStr);
    if (!isNaN(startObj.getTime())) {
      const newEnd = calculateEndDate(startObj, quickDuration);
      setEndDateStr(toLocalISOString(newEnd));
      if (fieldErrors.endDate) {
        setFieldErrors((prev) => ({ ...prev, endDate: '' }));
      }
    }
  };

  // При смене радио-длительности обновляем дату окончания
  const handleDurationRadioChange = (dur: QuickDuration) => {
    setQuickDuration(dur);
    const startObj = new Date(startDateStr);
    if (!isNaN(startObj.getTime())) {
      const newEnd = calculateEndDate(startObj, dur);
      setEndDateStr(toLocalISOString(newEnd));
      if (fieldErrors.endDate) {
        setFieldErrors((prev) => ({ ...prev, endDate: '' }));
      }
    }
  };

  const validate = () => {
    const errors: Record<string, string> = {};
    if (!selectedItemId) {
      errors.item = 'Пожалуйста, выберите предмет для продажи или загрузите новый';
    }
    if (!title.trim()) {
      errors.title = 'Укажите название аукциона';
    } else if (title.trim().length < 3) {
      errors.title = 'Название аукциона должно содержать не менее 3 символов';
    }
    if (!startAmount || startAmount < 100) {
      errors.startAmount = 'Стартовая цена должна быть не менее 100 Coins';
    }
    if (!minStep || minStep < 50) {
      errors.minStep = 'Минимальный шаг ставки должен быть не менее 50 Coins';
    }

    // Проверка даты начала: нельзя раньше чем через 1 час от текущего времени (только для модерации)
    const now = Date.now();
    const startTimestamp = new Date(startDateStr).getTime();
    const endTimestamp = new Date(endDateStr).getTime();

    if (isNaN(startTimestamp)) {
      errors.startDate = 'Укажите корректную дату и время старта';
    } else if (!isDraft && startTimestamp < now + 59 * 60 * 1000) {
      errors.startDate = 'Нельзя ставить начало лота раньше чем через 1 час от текущего времени';
    }

    if (isNaN(endTimestamp)) {
      errors.endDate = 'Укажите корректную дату и время окончания';
    } else if (endTimestamp <= startTimestamp) {
      errors.endDate = 'Время окончания должно быть позже времени начала';
    }

    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    setSubmitting(true);
    setError(null);
    try {
      const startIso = new Date(startDateStr).toISOString();
      const endIso = new Date(endDateStr).toISOString();
      const durationMin = Math.round(
        (new Date(endDateStr).getTime() - new Date(startDateStr).getTime()) / 60000
      );

      const payload = {
        item_id: selectedItemId!,
        title: title.trim(),
        category,
        short_text: shortText.trim(),
        text: text.trim() || shortText.trim() || title.trim(),
        start_amount: startAmount,
        min_step: minStep,
        duration_minutes: durationMin,
        started_at: startIso,
        ends_at: endIso,
        status: (isDraft ? 'draft' : 'pending_review') as 'draft' | 'pending_review',
        auto_activate: autoActivate,
      };

      if (editingLotId) {
        const updated = await updateLot(editingLotId, payload);
        cacheLot(updated);
        navigate(`/lot/${updated.id}`);
      } else {
        const created = await createLot(payload);
        cacheLot(created);
        navigate(`/lot/${created.id}`);
      }
    } catch (err) {
      setError(err);
    } finally {
      setSubmitting(false);
    }
  };

  const handleActivateEditingLot = async () => {
    if (!editingLotId) return;
    setActivating(true);
    setError(null);
    setActivateSuccess(null);
    try {
      await activateLot(editingLotId);
      setActivateSuccess('Лот успешно активирован! Переход на страницу открытых торгов...');
      setTimeout(() => navigate(`/lot/${editingLotId}`), 1200);
    } catch (err) {
      setError(err);
    } finally {
      setActivating(false);
    }
  };

  // Предпросмотр карточки лота
  const previewLot: Lot = {
    id: editingLotId || 99999,
    seller_id: user?.id || 'me',
    seller: {
      id: user?.id || 'me',
      name: user?.name || 'Вы (продавец)',
      avatar_id: user?.avatar_id,
      rating: 5.0,
    },
    item_id: selectedItemId || 1,
    status: isDraft ? 'draft' : 'active',
    category,
    title: title || 'Название лота',
    description: {
      text: text || 'Полное описание лота...',
      short_text: shortText || 'Краткое описание для карточки',
      media: [{ id: 'm1', type: 'image', order: 1 }],
      auto_activate: autoActivate,
    },
    start_amount: startAmount,
    current_bid: startAmount,
    min_step: minStep,
    finish_amount: null,
    bids_count: 0,
    winner: null,
    created_at: new Date().toISOString(),
    started_at: new Date(startDateStr).toISOString(),
    ends_at: new Date(endDateStr).toISOString(),
    finished_at: null,
  };

  return (
    <div className="space-y-6 pb-12">
      <div className="flex items-center justify-between">
        <Link
          to="/lots"
          className="inline-flex items-center gap-1.5 text-xs font-semibold text-slate-600 hover:text-teal-700 transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          <span>Назад в каталог</span>
        </Link>

        {editingLotId && (
          <button
            type="button"
            onClick={resetToCreateNew}
            className="inline-flex items-center gap-1 text-xs font-semibold text-slate-600 hover:text-teal-700 transition-colors cursor-pointer"
          >
            <RotateCcw className="w-3.5 h-3.5" />
            <span>Создать новый вместо редактирования</span>
          </button>
        )}
      </div>

      {/* Заголовок и селектор неначавшихся лотов */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-slate-900">
            {editingLotId ? `Редактировать лот #${editingLotId}` : 'Выставить лот на аукцион'}
          </h1>
          <p className="text-xs sm:text-sm text-slate-500">
            {editingLotId
              ? 'Измените параметры черновика или лота на модерации'
              : 'Заполните параметры лота. Предпросмотр обновляется в реальном времени.'}
          </p>
        </div>

        {/* Выбор из своих недоделанных / неначавшихся лотов */}
        {unstartedLots.length > 0 && (
          <div className="flex items-center gap-2">
            <span className="text-xs font-semibold text-slate-500 hidden md:inline">
              Выбрать лот:
            </span>
            <select
              value={editingLotId || '__new__'}
              onChange={(e) => {
                const val = e.target.value;
                if (val === '__new__') {
                  resetToCreateNew();
                } else {
                  const targetLot = unstartedLots.find((l) => l.id === Number(val));
                  if (targetLot) loadLotIntoForm(targetLot);
                }
              }}
              className="px-3 py-1.5 text-xs bg-slate-50 hover:bg-slate-100 border border-slate-200 text-slate-800 rounded-xl focus:outline-hidden focus:ring-1 focus:ring-teal-500 font-medium cursor-pointer"
            >
              <option value="__new__">+ Создать новый лот</option>
              <optgroup label="Неначавшиеся лоты (черновики и на проверке)">
                {unstartedLots.map((l) => (
                  <option key={l.id} value={l.id}>
                    #{l.id}: {l.title} ({l.status})
                  </option>
                ))}
              </optgroup>
            </select>
          </div>
        )}
      </div>

      {/* Баннер редактирования выбранного лота */}
      {editingLotId && (
        <div className="p-3.5 bg-amber-50 border border-amber-200 rounded-2xl flex flex-col sm:flex-row sm:items-center justify-between gap-3 shadow-2xs animate-in fade-in">
          <div className="flex items-center gap-2.5">
            <div className="p-2 bg-amber-500 text-white rounded-xl">
              <Edit3 className="w-4 h-4" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="text-xs font-bold text-amber-950">
                  Режим редактирования лота #{editingLotId}
                </span>
                <span className="text-[10px] font-bold px-2 py-0.5 rounded-full bg-amber-200/80 text-amber-900 uppercase">
                  {editingLotStatus}
                </span>
              </div>
              <p className="text-[11px] text-amber-800 mt-0.5">
                Вы можете обновить текст, предмет, стартовую цену, шаг ставки и расписание.
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            {editingLotStatus === 'approved' && (
              <button
                type="button"
                onClick={handleActivateEditingLot}
                disabled={activating}
                className="flex items-center gap-1.5 px-3.5 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-bold rounded-xl shadow-xs transition-colors cursor-pointer disabled:opacity-50"
              >
                {activating ? (
                  <Loader2 className="w-3.5 h-3.5 animate-spin" />
                ) : (
                  <Play className="w-3.5 h-3.5 fill-current" />
                )}
                <span>Активировать торги сейчас</span>
              </button>
            )}
            <button
              type="button"
              onClick={resetToCreateNew}
              className="text-xs text-amber-900 hover:underline font-semibold"
            >
              Сбросить
            </button>
          </div>
        </div>
      )}

      {activateSuccess && (
        <div className="p-3 bg-emerald-50 border border-emerald-200 rounded-xl text-emerald-900 text-xs font-semibold flex items-center gap-2">
          <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
          <span>{activateSuccess}</span>
        </div>
      )}

      {error ? <ErrorAlert error={error} onDismiss={() => setError(null)} /> : null}

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* Форма создания / редактирования */}
        <form
          onSubmit={handleSubmit}
          noValidate
          className="lg:col-span-7 bg-white rounded-2xl border border-slate-200/80 p-5 sm:p-6 shadow-xs space-y-4"
        >
          {/* Выбор предмета: select, последний пункт — добавление предмета */}
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className="block text-xs font-bold text-slate-700">
                Предмет для продажи
              </label>
              <Link
                to="/profile#items"
                className="inline-flex items-center gap-1 text-xs font-bold text-teal-700 hover:text-teal-800 hover:underline"
              >
                <Plus className="w-3.5 h-3.5" />
                <span>Добавить предмет</span>
              </Link>
            </div>

            <select
              value={selectedItemId || ''}
              onChange={(e) => {
                const val = e.target.value;
                if (val === '__add_new__') {
                  navigate('/profile#items');
                  return;
                }
                const id = Number(val);
                setSelectedItemId(id);
                if (fieldErrors.item) {
                  setFieldErrors((prev) => ({ ...prev, item: '' }));
                }
                const found = items.find((i) => i.id === id);
                if (found && found.metadata?.title) {
                  setTitle(found.metadata.title as string);
                }
              }}
              className={`w-full px-3.5 py-2.5 text-sm bg-slate-50 border rounded-xl focus:outline-hidden focus:ring-2 ${
                fieldErrors.item
                  ? 'border-rose-400 bg-rose-50/20 focus:ring-rose-400/20'
                  : 'border-slate-200 focus:ring-teal-500/20'
              }`}
            >
              <option value="" disabled>
                {items.length === 0 ? '— Нет загруженных предметов —' : '— Выберите предмет —'}
              </option>
              {items.map((it) => (
                <option key={it.id} value={it.id}>
                  {(it.metadata?.title as string) || `Предмет #${it.id}`} ({it.type})
                </option>
              ))}
              <option value="__add_new__" className="text-teal-700 font-bold bg-teal-50">
                + Добавить новый предмет в личном кабинете...
              </option>
            </select>

            {fieldErrors.item ? (
              <p className="mt-1 text-xs text-rose-600 font-medium">{fieldErrors.item}</p>
            ) : null}
          </div>

          {/* Название */}
          <div>
            <label className="block text-xs font-bold text-slate-700 mb-1">
              Название аукциона
            </label>
            <input
              type="text"
              value={title}
              onChange={(e) => {
                setTitle(e.target.value);
                if (fieldErrors.title) {
                  setFieldErrors((prev) => ({ ...prev, title: '' }));
                }
              }}
              placeholder="Например: 3D-модель боевого меха или Исходник арта"
              className={`w-full px-3.5 py-2.5 text-sm bg-slate-50 border rounded-xl focus:outline-hidden focus:ring-2 ${
                fieldErrors.title
                  ? 'border-rose-400 bg-rose-50/20 focus:ring-rose-400/20'
                  : 'border-slate-200 focus:ring-teal-500/20'
              }`}
            />
            {fieldErrors.title ? (
              <p className="mt-1 text-xs text-rose-600 font-medium">{fieldErrors.title}</p>
            ) : null}
          </div>

          {/* Категория: определяется автоматически по типу предмета */}
          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="block text-xs font-bold text-slate-700">Категория</label>
              <span className="text-[11px] text-slate-400">
                Определяется автоматически по типу предмета
              </span>
            </div>
            <input
              type="text"
              readOnly
              disabled
              value={
                selectedItem
                  ? `${category} (${selectedItem.type})`
                  : '— Выберите предмет выше —'
              }
              className="w-full px-3.5 py-2.5 text-sm bg-slate-100 text-slate-600 border border-slate-200 rounded-xl cursor-not-allowed select-none font-medium"
            />
          </div>

          {/* Краткое описание */}
          <div>
            <label className="block text-xs font-bold text-slate-700 mb-1">
              Краткое описание (для карточки)
            </label>
            <input
              type="text"
              value={shortText}
              onChange={(e) => setShortText(e.target.value)}
              placeholder="1-2 предложения, привлекающие участников"
              className="w-full px-3.5 py-2.5 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-hidden focus:ring-2 focus:ring-teal-500/20"
            />
          </div>

          {/* Полное описание */}
          <div>
            <label className="block text-xs font-bold text-slate-700 mb-1">
              Подробное описание
            </label>
            <textarea
              rows={3}
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder="Технические особенности, состав архива, лицензия..."
              className="w-full px-3.5 py-2.5 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-hidden focus:ring-2 focus:ring-teal-500/20"
            />
          </div>

          {/* Цены и шаг */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
            <div>
              <label className="block text-xs font-bold text-slate-700 mb-1">
                Стартовая цена (🪙 Coins)
              </label>
              <input
                type="number"
                min={100}
                step={100}
                value={startAmount}
                onChange={(e) => {
                  const val = Number(e.target.value);
                  setStartAmount(val);
                  if (fieldErrors.startAmount && val >= 100) {
                    setFieldErrors((prev) => ({ ...prev, startAmount: '' }));
                  }
                }}
                className={`w-full px-3.5 py-2.5 text-sm font-bold bg-slate-50 border rounded-xl focus:outline-hidden focus:ring-2 ${
                  fieldErrors.startAmount
                    ? 'border-rose-400 bg-rose-50/20 focus:ring-rose-400/20'
                    : 'border-slate-200 focus:ring-teal-500/20'
                }`}
              />
              {fieldErrors.startAmount ? (
                <p className="mt-1 text-xs text-rose-600 font-medium">{fieldErrors.startAmount}</p>
              ) : null}
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 mb-1">
                Минимальный шаг (🪙 Coins)
              </label>
              <input
                type="number"
                min={50}
                step={50}
                value={minStep}
                onChange={(e) => {
                  const val = Number(e.target.value);
                  setMinStep(val);
                  if (fieldErrors.minStep && val >= 50) {
                    setFieldErrors((prev) => ({ ...prev, minStep: '' }));
                  }
                }}
                className={`w-full px-3.5 py-2.5 text-sm font-bold bg-slate-50 border rounded-xl focus:outline-hidden focus:ring-2 ${
                  fieldErrors.minStep
                    ? 'border-rose-400 bg-rose-50/20 focus:ring-rose-400/20'
                    : 'border-slate-200 focus:ring-teal-500/20'
                }`}
              />
              {fieldErrors.minStep ? (
                <p className="mt-1 text-xs text-rose-600 font-medium">{fieldErrors.minStep}</p>
              ) : null}
            </div>
          </div>

          {/* Дата начала торгов */}
          <div className="pt-2">
            <label className="block text-xs font-bold text-slate-700 mb-1 flex items-center gap-1.5">
              <Calendar className="w-3.5 h-3.5 text-teal-600" />
              <span>Дата и время старта аукциона</span>
            </label>
            <input
              type="datetime-local"
              value={startDateStr}
              onChange={(e) => handleStartDateChange(e.target.value)}
              className={`w-full px-3.5 py-2.5 text-sm bg-slate-50 border rounded-xl focus:outline-hidden focus:ring-2 ${
                fieldErrors.startDate
                  ? 'border-rose-400 bg-rose-50/20 focus:ring-rose-400/20'
                  : 'border-slate-200 focus:ring-teal-500/20'
              }`}
            />
            {fieldErrors.startDate ? (
              <p className="mt-1 text-xs text-rose-600 font-medium">{fieldErrors.startDate}</p>
            ) : null}
          </div>

          {/* Быстрый выбор длительности */}
          <div>
            <label className="block text-xs font-bold text-slate-700 mb-1 flex items-center gap-1.5">
              <Clock className="w-3.5 h-3.5 text-teal-600" />
              <span>Длительность торгов</span>
            </label>
            <div className="grid grid-cols-3 gap-2">
              {[
                { id: 'hour', label: '1 час' },
                { id: 'day', label: '24 часа (1 день)' },
                { id: 'week', label: '7 дней (неделя)' },
              ].map((opt) => (
                <button
                  key={opt.id}
                  type="button"
                  onClick={() => handleDurationRadioChange(opt.id as QuickDuration)}
                  className={`px-3 py-2 text-xs font-semibold rounded-xl border transition-colors cursor-pointer text-center ${
                    quickDuration === opt.id
                      ? 'border-teal-600 bg-teal-50 text-teal-800 font-bold'
                      : 'border-slate-200 bg-slate-50 text-slate-600 hover:bg-slate-100'
                  }`}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          </div>

          {/* Дата окончания торгов */}
          <div>
            <label className="block text-xs font-bold text-slate-700 mb-1">
              Дата и время окончания
            </label>
            <input
              type="datetime-local"
              value={endDateStr}
              onChange={(e) => {
                setEndDateStr(e.target.value);
                if (fieldErrors.endDate) {
                  setFieldErrors((prev) => ({ ...prev, endDate: '' }));
                }
              }}
              className={`w-full px-3.5 py-2.5 text-sm bg-slate-50 border rounded-xl focus:outline-hidden focus:ring-2 ${
                fieldErrors.endDate
                  ? 'border-rose-400 bg-rose-50/20 focus:ring-rose-400/20'
                  : 'border-slate-200 focus:ring-teal-500/20'
              }`}
            />
            {fieldErrors.endDate ? (
              <p className="mt-1 text-xs text-rose-600 font-medium">{fieldErrors.endDate}</p>
            ) : null}
          </div>

          {/* Настройки: Галочка Draft и Галочка Авто-запуска */}
          <div className="p-4 bg-slate-50 border border-slate-200 rounded-xl space-y-3 pt-3">
            <label className="flex items-start gap-2.5 cursor-pointer select-none">
              <input
                type="checkbox"
                checked={isDraft}
                onChange={(e) => setIsDraft(e.target.checked)}
                className="mt-0.5 w-4 h-4 text-teal-600 rounded border-slate-300 focus:ring-teal-500 cursor-pointer"
              />
              <div>
                <span className="text-xs font-bold text-slate-800">
                  Сохранить как черновик (Draft)
                </span>
                <p className="text-[11px] text-slate-500">
                  {isDraft
                    ? 'Лот останется в черновиках. Вы сможете доработать и отправить его позже.'
                    : 'Лот сразу поступит на модерацию (Pending Review) для проверки администратором.'}
                </p>
              </div>
            </label>

            <div className="border-t border-slate-200/80 pt-3">
              <label className="flex items-start gap-2.5 cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={autoActivate}
                  onChange={(e) => setAutoActivate(e.target.checked)}
                  className="mt-0.5 w-4 h-4 text-teal-600 rounded border-slate-300 focus:ring-teal-500 cursor-pointer"
                />
                <div>
                  <span className="text-xs font-bold text-slate-800">
                    Автоматический запуск торгов (auto_activate)
                  </span>
                  <p className="text-[11px] text-slate-500">
                    {autoActivate
                      ? 'Торги начнутся автоматически после одобрения лота в запланированное время старта.'
                      : 'После одобрения лот перейдёт в статус «Approved», и вы сможете нажать кнопку «Активировать» вручную.'}
                  </p>
                </div>
              </label>
            </div>
          </div>

          {/* Кнопка отправки */}
          <div className="pt-2">
            <button
              type="submit"
              disabled={submitting}
              className="w-full py-3 bg-teal-600 hover:bg-teal-700 active:bg-teal-800 text-white font-bold text-sm rounded-xl shadow-xs transition-colors flex items-center justify-center gap-2 cursor-pointer disabled:opacity-50"
            >
              {submitting ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  <span>Сохранение лота...</span>
                </>
              ) : (
                <>
                  <Gavel className="w-4 h-4" />
                  <span>
                    {editingLotId
                      ? `Сохранить изменения в лоте #${editingLotId}`
                      : isDraft
                      ? 'Сохранить как черновик'
                      : 'Отправить лот на модерацию'}
                  </span>
                </>
              )}
            </button>
          </div>
        </form>

        {/* Интерактивный предпросмотр карточки лота */}
        <div className="lg:col-span-5 space-y-3 sticky top-20">
          <div className="flex items-center justify-between">
            <h3 className="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1.5">
              <Eye className="w-3.5 h-3.5" />
              <span>Предпросмотр карточки</span>
            </h3>
            <span className="text-[11px] text-slate-400">
              {isDraft ? 'Статус: Черновик' : 'Статус: Модерация'}
            </span>
          </div>

          <div className="pointer-events-none">
            <LotCard lot={previewLot} interactive={false} />
          </div>

          <div className="p-3 bg-slate-50 border border-slate-200/80 rounded-xl text-[11px] text-slate-500 space-y-1">
            <div className="font-semibold text-slate-700">Информация о публикации:</div>
            <div>
              • Режим авто-запуска:{' '}
              <strong className={autoActivate ? 'text-teal-700' : 'text-slate-700'}>
                {autoActivate ? 'Включён (автоматически)' : 'Выключен (ручной старт)'}
              </strong>
            </div>
            <div>
              • Статус сохранения:{' '}
              <strong className={isDraft ? 'text-amber-700' : 'text-blue-700'}>
                {isDraft ? 'Черновик (Draft)' : 'На проверку (Pending Review)'}
              </strong>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
