import { useState, useEffect } from 'react';
import { useNavigate, useSearchParams, Link } from 'react-router';
import { Gavel, Loader2, Plus, Eye, ArrowLeft, Calendar, Clock, AlertCircle } from 'lucide-react';
import { useAuth } from '~/hooks/useAuth';
import { listUserItems } from '~/api/items';
import { createLot } from '~/api/lots';
import type { Item, Lot } from '~/api/types';
import { LotCard } from '~/components/lot/LotCard';
import { ErrorAlert } from '~/components/feedback/ErrorAlert';

// 6 категорий строго в соответствии с Go itemDomain.go
const CATEGORIES = [
  '3D Модели',
  'Цифровой Арт',
  'Аудио',
  'Видео',
  'Документы',
  'Архивы',
];

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
  // Если вдруг завтра в 08:00 меньше чем через час от текущего момента
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
  const [searchParams] = useSearchParams();
  const preselectedItemId = searchParams.get('item_id');

  const [items, setItems] = useState<Item[]>([]);
  const [selectedItemId, setSelectedItemId] = useState<number | null>(
    preselectedItemId ? Number(preselectedItemId) : null
  );

  const [title, setTitle] = useState('');
  const [category, setCategory] = useState(CATEGORIES[0]);
  const [shortText, setShortText] = useState('');
  const [text, setText] = useState('');
  const [startAmount, setStartAmount] = useState(5000);
  const [minStep, setMinStep] = useState(250);

  // Даты старта и окончания
  const [startDateStr, setStartDateStr] = useState(() => toLocalISOString(getDefaultStartDate()));
  const [quickDuration, setQuickDuration] = useState<QuickDuration>('day');
  const [endDateStr, setEndDateStr] = useState(() =>
    toLocalISOString(calculateEndDate(getDefaultStartDate(), 'day'))
  );

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<unknown | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      navigate('/login');
      return;
    }
    if (isAuthenticated) {
      listUserItems()
        .then((data) => {
          setItems(data);
          if (!selectedItemId && data.length > 0) {
            setSelectedItemId(data[0].id);
            if (data[0].metadata?.title) {
              setTitle(data[0].metadata.title as string);
            }
          }
        })
        .catch(() => {});
    }
  }, [authLoading, isAuthenticated, navigate, selectedItemId]);

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

    // Проверка даты начала: нельзя раньше чем через час от текущего времени
    const now = Date.now();
    const startTimestamp = new Date(startDateStr).getTime();
    const endTimestamp = new Date(endDateStr).getTime();

    if (isNaN(startTimestamp)) {
      errors.startDate = 'Укажите корректную дату и время старта';
    } else if (startTimestamp < now + 59 * 60 * 1000) {
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

      const created = await createLot({
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
      });
      navigate(`/lot/${created.id}`);
    } catch (err) {
      setError(err);
    } finally {
      setSubmitting(false);
    }
  };

  // Предпросмотр карточки лота: не кликается и не имеет таймера (interactive={false})
  const previewLot: Lot = {
    id: 99999,
    seller_id: user?.id || 'me',
    seller: {
      id: user?.id || 'me',
      name: user?.name || 'Вы (продавец)',
      avatar_id: user?.avatar_id,
      rating: 5.0,
    },
    item_id: selectedItemId || 1,
    status: 'active',
    category,
    title: title || 'Название лота',
    description: {
      text: text || 'Полное описание лота...',
      short_text: shortText || 'Краткое описание для карточки',
      media: [{ id: 'm1', type: 'image', order: 1 }],
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
      <div className="flex items-center gap-2">
        <Link
          to="/lots"
          className="inline-flex items-center gap-1.5 text-xs font-semibold text-slate-600 hover:text-teal-700 transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          <span>Назад в каталог</span>
        </Link>
      </div>

      <div className="flex flex-col gap-1">
        <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-slate-900">
          Выставить лот на аукцион
        </h1>
        <p className="text-xs sm:text-sm text-slate-500">
          Заполните параметры лота. Предпросмотр обновляется в реальном времени.
        </p>
      </div>

      {error ? <ErrorAlert error={error} onDismiss={() => setError(null)} /> : null}

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* Форма создания */}
        <form
          onSubmit={handleSubmit}
          noValidate
          className="lg:col-span-7 bg-white rounded-2xl border border-slate-200/80 p-5 sm:p-6 shadow-xs space-y-4"
        >
          {/* Выбор предмета: ВСЕГДА select, последний пункт — добавление предмета */}
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

          {/* Категория (6 категорий строго из itemDomain) */}
          <div>
            <label className="block text-xs font-bold text-slate-700 mb-1">Категория</label>
            <select
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              className="w-full px-3.5 py-2.5 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-hidden focus:ring-2 focus:ring-teal-500/20"
            >
              {CATEGORIES.map((cat) => (
                <option key={cat} value={cat}>
                  {cat}
                </option>
              ))}
            </select>
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

          {/* Расписание аукциона: Дата старта + Радио выбор длительности + Дата окончания */}
          <div className="pt-2 border-t border-slate-100 space-y-4">
            <h3 className="text-xs font-bold text-slate-800 uppercase tracking-wider flex items-center gap-1.5">
              <Calendar className="w-3.5 h-3.5 text-teal-600" />
              <span>Время проведения аукциона</span>
            </h3>

            {/* Выбор даты и времени старта */}
            <div>
              <div className="flex items-center justify-between mb-1">
                <label className="block text-xs font-bold text-slate-700">
                  Дата и время начала торгов
                </label>
                <span className="text-[11px] text-slate-400">минимум через 1 час</span>
              </div>
              <input
                type="datetime-local"
                value={startDateStr}
                onChange={(e) => handleStartDateChange(e.target.value)}
                className={`w-full px-3.5 py-2.5 text-sm bg-slate-50 border rounded-xl focus:outline-hidden focus:ring-2 font-medium ${
                  fieldErrors.startDate
                    ? 'border-rose-400 bg-rose-50/20 focus:ring-rose-400/20'
                    : 'border-slate-200 focus:ring-teal-500/20'
                }`}
              />
              {fieldErrors.startDate ? (
                <p className="mt-1 text-xs text-rose-600 font-medium flex items-center gap-1">
                  <AlertCircle className="w-3 h-3" />
                  {fieldErrors.startDate}
                </p>
              ) : null}
            </div>

            {/* Радио-переключатели быстрой длительности (Час, День, Неделя) */}
            <div>
              <label className="block text-xs font-bold text-slate-700 mb-1.5">
                Длительность торгов (быстрый выбор)
              </label>
              <div className="grid grid-cols-3 gap-2">
                <label
                  className={`flex items-center justify-center gap-2 p-2.5 rounded-xl border text-xs font-semibold cursor-pointer transition-all ${
                    quickDuration === 'hour'
                      ? 'border-teal-600 bg-teal-50/70 text-teal-900 ring-2 ring-teal-600/20'
                      : 'border-slate-200 hover:bg-slate-50 text-slate-700'
                  }`}
                >
                  <input
                    type="radio"
                    name="quickDuration"
                    value="hour"
                    checked={quickDuration === 'hour'}
                    onChange={() => handleDurationRadioChange('hour')}
                    className="sr-only"
                  />
                  <span>1 час</span>
                </label>

                <label
                  className={`flex items-center justify-center gap-2 p-2.5 rounded-xl border text-xs font-semibold cursor-pointer transition-all ${
                    quickDuration === 'day'
                      ? 'border-teal-600 bg-teal-50/70 text-teal-900 ring-2 ring-teal-600/20'
                      : 'border-slate-200 hover:bg-slate-50 text-slate-700'
                  }`}
                >
                  <input
                    type="radio"
                    name="quickDuration"
                    value="day"
                    checked={quickDuration === 'day'}
                    onChange={() => handleDurationRadioChange('day')}
                    className="sr-only"
                  />
                  <span>1 день (24 ч)</span>
                </label>

                <label
                  className={`flex items-center justify-center gap-2 p-2.5 rounded-xl border text-xs font-semibold cursor-pointer transition-all ${
                    quickDuration === 'week'
                      ? 'border-teal-600 bg-teal-50/70 text-teal-900 ring-2 ring-teal-600/20'
                      : 'border-slate-200 hover:bg-slate-50 text-slate-700'
                  }`}
                >
                  <input
                    type="radio"
                    name="quickDuration"
                    value="week"
                    checked={quickDuration === 'week'}
                    onChange={() => handleDurationRadioChange('week')}
                    className="sr-only"
                  />
                  <span>1 неделя</span>
                </label>
              </div>
            </div>

            {/* Выбор даты и времени конца */}
            <div>
              <div className="flex items-center justify-between mb-1">
                <label className="block text-xs font-bold text-slate-700">
                  Дата и время окончания торгов
                </label>
                <span className="text-[11px] text-slate-400">автоматически или вручную</span>
              </div>
              <input
                type="datetime-local"
                value={endDateStr}
                onChange={(e) => {
                  setEndDateStr(e.target.value);
                  if (fieldErrors.endDate) {
                    setFieldErrors((prev) => ({ ...prev, endDate: '' }));
                  }
                }}
                className={`w-full px-3.5 py-2.5 text-sm bg-slate-50 border rounded-xl focus:outline-hidden focus:ring-2 font-medium ${
                  fieldErrors.endDate
                    ? 'border-rose-400 bg-rose-50/20 focus:ring-rose-400/20'
                    : 'border-slate-200 focus:ring-teal-500/20'
                }`}
              />
              {fieldErrors.endDate ? (
                <p className="mt-1 text-xs text-rose-600 font-medium flex items-center gap-1">
                  <AlertCircle className="w-3 h-3" />
                  {fieldErrors.endDate}
                </p>
              ) : null}
            </div>
          </div>

          <button
            type="submit"
            disabled={submitting || !title.trim() || !selectedItemId}
            className="w-full py-3 bg-teal-600 hover:bg-teal-700 text-white font-bold text-sm rounded-xl shadow-md transition-colors flex items-center justify-center gap-2 disabled:opacity-50 mt-4 cursor-pointer"
          >
            {submitting ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                <span>Запуск аукциона...</span>
              </>
            ) : (
              <>
                <Gavel className="w-4 h-4" />
                <span>Запустить аукцион</span>
              </>
            )}
          </button>
        </form>

        {/* Неинтерактивный предпросмотр (без таймера, не кликается) */}
        <div className="lg:col-span-5 space-y-3 lg:sticky lg:top-20">
          <div className="flex items-center gap-1.5 text-xs font-bold text-slate-500 uppercase tracking-wider">
            <Eye className="w-3.5 h-3.5" />
            <span>Предпросмотр карточки в каталоге</span>
          </div>

          <LotCard lot={previewLot} interactive={false} />

          <p className="text-[11px] text-slate-400 text-center leading-relaxed">
            Так карточка будет выглядеть в общем каталоге. Во время предпросмотра переход по клику и таймер отключены.
          </p>
        </div>
      </div>
    </div>
  );
}
