import React from 'react';
import {
  Search,
  RotateCcw,
  SlidersHorizontal,
  FileBox,
  Box,
  Music,
  Video,
  Archive,
  FileText,
  Palette,
  X,
} from 'lucide-react';
import type { LotFilter } from '~/api/types';

interface LotFiltersProps {
  filter: LotFilter;
  onChange: (filter: LotFilter) => void;
  onReset: () => void;
}

export const CATEGORIES = [
  { label: 'Все файлы', value: '', icon: FileBox },
  { label: 'Изображения', value: 'image', icon: Palette },
  { label: '3D Модели', value: '3d', icon: Box },
  { label: 'Аудио', value: 'audio', icon: Music },
  { label: 'Видео', value: 'video', icon: Video },
  { label: 'Документы', value: 'document', icon: FileText },
  { label: 'Архивы', value: 'archive', icon: Archive },
];

const SORTS = [
  { value: 'newest', label: 'Сначала новые' },
  { value: 'ending_soon', label: 'Скоро завершаются' },
  { value: 'popular', label: 'По популярности' },
  { value: 'price_asc', label: 'Сначала дешевле' },
  { value: 'price_desc', label: 'Сначала дороже' },
];

export function LotFilters({ filter, onChange, onReset }: LotFiltersProps) {
  const hasActiveFilters =
    Boolean(filter.query) ||
    Boolean(filter.category) ||
    (filter.status && filter.status !== 'active') ||
    (filter.sort && filter.sort !== 'newest');

  return (
    <div className="bg-white rounded-2xl border border-slate-200/80 p-3.5 sm:p-4.5 shadow-xs space-y-3.5 transition-all">
      {/* Верхняя строка: Поиск и Сегментированный переключатель статуса */}
      <div className="flex flex-col md:flex-row items-stretch md:items-center gap-3">
        {/* Поле поиска */}
        <div className="relative flex-1">
          <Search className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none" />
          <input
            type="text"
            placeholder="Поиск по названию файла, формату или автору..."
            value={filter.query || ''}
            onChange={(e) => onChange({ ...filter, query: e.target.value })}
            className="w-full pl-10 pr-9 py-2 text-xs sm:text-sm bg-slate-50 hover:bg-slate-50/80 border border-slate-200 rounded-xl focus:bg-white focus:outline-hidden focus:ring-2 focus:ring-teal-500/25 focus:border-teal-500 transition-all placeholder:text-slate-400"
          />
          {filter.query && (
            <button
              type="button"
              onClick={() => onChange({ ...filter, query: '' })}
              className="absolute right-2.5 top-1/2 -translate-y-1/2 p-1 text-slate-400 hover:text-slate-600 rounded-full hover:bg-slate-200"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          )}
        </div>

        {/* Стильный сегментированный переключатель статуса */}
        <div className="flex items-center p-1 bg-slate-100 rounded-xl self-start md:self-auto shrink-0 shadow-2xs overflow-x-auto max-w-full">
          <button
            type="button"
            onClick={() => onChange({ ...filter, status: 'active' })}
            className={`px-2.5 sm:px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer whitespace-nowrap ${
              filter.status === 'active'
                ? 'bg-white text-teal-900 shadow-xs ring-1 ring-slate-200/60'
                : 'text-slate-500 hover:text-slate-900'
            }`}
          >
            Активные торги
          </button>
          <button
            type="button"
            onClick={() => onChange({ ...filter, status: 'upcoming' })}
            className={`px-2.5 sm:px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer whitespace-nowrap ${
              filter.status === 'upcoming'
                ? 'bg-white text-amber-900 shadow-xs ring-1 ring-slate-200/60 font-bold'
                : 'text-slate-500 hover:text-amber-700'
            }`}
          >
            Скоро начнутся
          </button>
          <button
            type="button"
            onClick={() => onChange({ ...filter, status: '' })}
            className={`px-2.5 sm:px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer whitespace-nowrap ${
              !filter.status
                ? 'bg-white text-teal-900 shadow-xs ring-1 ring-slate-200/60'
                : 'text-slate-500 hover:text-slate-900'
            }`}
          >
            Все
          </button>
          <button
            type="button"
            onClick={() => onChange({ ...filter, status: 'finished' })}
            className={`px-2.5 sm:px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer whitespace-nowrap ${
              filter.status === 'finished'
                ? 'bg-white text-teal-900 shadow-xs ring-1 ring-slate-200/60'
                : 'text-slate-500 hover:text-slate-900'
            }`}
          >
            Завершённые
          </button>
        </div>
      </div>

      {/* Нижняя строка: Интерактивные чипсы категорий и сортировка */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-2.5 pt-3 border-t border-slate-100">
        {/* Чипсы категорий: аккуратный скролл со скрытием под разделителем */}
        <div className="relative flex-1 min-w-0">
          <div className="flex items-center gap-1.5 overflow-x-auto no-scrollbar pr-2 py-0.5">
            {CATEGORIES.map((cat) => {
              const Icon = cat.icon;
              const isSelected = (filter.category || '') === cat.value;

              return (
                <button
                  key={cat.label}
                  type="button"
                  onClick={() => onChange({ ...filter, category: cat.value })}
                  className={`flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-medium whitespace-nowrap transition-all cursor-pointer shrink-0 ${
                    isSelected
                      ? 'bg-teal-600 text-white shadow-xs font-semibold'
                      : 'bg-slate-50 hover:bg-slate-100 text-slate-600 border border-slate-200/70'
                  }`}
                >
                  <Icon className={`w-3.5 h-3.5 ${isSelected ? 'text-teal-100' : 'text-slate-400'}`} />
                  <span>{cat.label}</span>
                </button>
              );
            })}
          </div>
        </div>

        {/* Вертикальная перегородка (черточка-разделитель) */}
        <div className="hidden md:block w-px h-6 bg-slate-200 shrink-0 mx-1" />

        {/* Блок сортировки */}
        <div className="flex items-center justify-end gap-2 shrink-0 pt-2 md:pt-0 border-t md:border-t-0 border-slate-200/80">
          <div className="flex items-center gap-1.5 px-2.5 py-1 bg-slate-50 border border-slate-200/80 rounded-xl text-xs font-medium text-slate-700">
            <SlidersHorizontal className="w-3.5 h-3.5 text-slate-400" />
            <select
              value={filter.sort || 'newest'}
              onChange={(e) => onChange({ ...filter, sort: e.target.value })}
              className="bg-transparent focus:outline-hidden font-medium text-slate-700 cursor-pointer text-xs"
            >
              {SORTS.map((s) => (
                <option key={s.value} value={s.value}>
                  {s.label}
                </option>
              ))}
            </select>
          </div>

          {/* Кнопка сброса фильтров */}
          {hasActiveFilters && (
            <button
              type="button"
              onClick={onReset}
              title="Сбросить фильтры"
              className="flex items-center gap-1 px-2.5 py-1.5 text-xs text-rose-600 hover:text-rose-700 bg-rose-50 hover:bg-rose-100/70 rounded-xl font-medium transition-colors cursor-pointer"
            >
              <RotateCcw className="w-3.5 h-3.5" />
              <span className="hidden sm:inline">Сбросить</span>
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
