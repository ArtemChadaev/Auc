import { useState } from 'react';
import { Link } from 'react-router';
import {
  Package,
  Plus,
  Gavel,
  Eye,
  Box,
  Music,
  Image as ImageIcon,
  Archive,
  FileText,
  FileCode,
  HardDrive,
} from 'lucide-react';
import type { Item, ItemType } from '~/api/types';
import { AddItemForm } from './AddItemForm';
import { ItemViewerModal } from './ItemViewerModal';

interface MyItemsProps {
  items: Item[];
  onItemAdded: (item: Item) => void;
  onItemRemoved?: (itemId: number) => void;
}

export function MyItems({ items, onItemAdded, onItemRemoved }: MyItemsProps) {
  const [showAddModal, setShowAddModal] = useState(false);
  const [selectedItemForView, setSelectedItemForView] = useState<Item | null>(null);

  const getItemIcon = (type: ItemType) => {
    switch (type) {
      case '3d':
        return <Box className="w-5 h-5 text-indigo-600" />;
      case 'audio':
        return <Music className="w-5 h-5 text-amber-600" />;
      case 'image':
        return <ImageIcon className="w-5 h-5 text-emerald-600" />;
      case 'archive':
        return <Archive className="w-5 h-5 text-rose-600" />;
      case 'document':
        return <FileText className="w-5 h-5 text-blue-600" />;
      default:
        return <FileCode className="w-5 h-5 text-slate-600" />;
    }
  };

  const getTypeBadgeClass = (type: ItemType) => {
    switch (type) {
      case '3d':
        return 'bg-indigo-50 text-indigo-700 border-indigo-200';
      case 'audio':
        return 'bg-amber-50 text-amber-700 border-amber-200';
      case 'image':
        return 'bg-emerald-50 text-emerald-700 border-emerald-200';
      case 'archive':
        return 'bg-rose-50 text-rose-700 border-rose-200';
      case 'document':
        return 'bg-blue-50 text-blue-700 border-blue-200';
      default:
        return 'bg-slate-100 text-slate-700 border-slate-200';
    }
  };

  return (
    <div className="bg-white rounded-2xl border border-slate-200/80 p-5 shadow-xs space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 font-bold text-slate-900 text-sm sm:text-base">
          <Package className="w-4 h-4 text-teal-600" />
          <span>Мои предметы</span>
          <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-slate-100 text-slate-600">
            {items.length}
          </span>
        </div>

        <button
          type="button"
          onClick={() => setShowAddModal(true)}
          className="flex items-center gap-1.5 px-3 py-1.5 bg-teal-50 hover:bg-teal-100 active:bg-teal-200 text-teal-800 text-xs font-bold rounded-xl transition-colors shadow-2xs"
        >
          <Plus className="w-3.5 h-3.5" />
          <span>Загрузить предмет</span>
        </button>
      </div>

      {items.length === 0 ? (
        <div className="text-center py-8 px-4 rounded-xl border border-dashed border-slate-200 bg-slate-50/50">
          <Package className="w-8 h-8 text-slate-300 mx-auto mb-2" />
          <p className="text-xs font-semibold text-slate-600">У вас пока нет загруженных предметов</p>
          <p className="text-[11px] text-slate-400 mt-0.5">
            Загрузите свой цифровой файл (3D, аудио, арт, архив), чтобы выставить его на торги.
          </p>
          <button
            type="button"
            onClick={() => setShowAddModal(true)}
            className="mt-3 px-3.5 py-1.5 bg-teal-600 hover:bg-teal-700 text-white text-xs font-bold rounded-xl shadow-xs transition-colors inline-flex items-center gap-1"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>Загрузить первый предмет</span>
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {items.map((item) => {
            const meta = item.metadata || {};
            const title = (meta.title as string) || (meta.filename as string) || `Предмет #${item.id}`;
            const fileSize = (meta.file_size as string) || '';
            const fileFormat = (meta.file_format as string) || item.type.toUpperCase();

            return (
              <div
                key={item.id}
                className="p-3.5 rounded-xl border border-slate-200/80 bg-slate-50/50 hover:bg-white hover:border-teal-500/40 hover:shadow-sm transition-all flex flex-col justify-between gap-3 group"
              >
                {/* Карточка предмета */}
                <div
                  className="cursor-pointer space-y-2.5"
                  onClick={() => setSelectedItemForView(item)}
                >
                  <div className="flex items-start gap-3">
                    <div className="w-10 h-10 rounded-xl bg-white border border-slate-200 flex items-center justify-center shrink-0 shadow-2xs group-hover:scale-105 transition-transform">
                      {getItemIcon(item.type)}
                    </div>
                    <div className="min-w-0 flex-1">
                      <h4 className="text-xs font-bold text-slate-900 truncate group-hover:text-teal-700 transition-colors">
                        {title}
                      </h4>
                      <div className="flex items-center gap-1.5 mt-1">
                        <span
                          className={`text-[9px] font-bold uppercase px-1.5 py-0.2 rounded border ${getTypeBadgeClass(
                            item.type
                          )}`}
                        >
                          {item.type}
                        </span>
                        <span className="text-[10px] text-slate-400 font-mono">#{item.id}</span>
                      </div>
                    </div>
                  </div>

                  {/* Мета-бейдж (размер и формат) */}
                  {(fileSize || fileFormat) && (
                    <div className="flex items-center gap-2 text-[10px] text-slate-500 bg-white/80 px-2 py-1 rounded-lg border border-slate-100">
                      {fileSize && (
                        <span className="flex items-center gap-1">
                          <HardDrive className="w-3 h-3 text-slate-400" />
                          {fileSize}
                        </span>
                      )}
                      {fileSize && fileFormat && <span className="text-slate-300">•</span>}
                      {fileFormat && <span className="font-semibold text-slate-700">{fileFormat}</span>}
                    </div>
                  )}
                </div>

                {/* Действия: Просмотр и Создать лот */}
                <div className="grid grid-cols-2 gap-2 pt-1 border-t border-slate-200/60">
                  <button
                    type="button"
                    onClick={() => setSelectedItemForView(item)}
                    className="py-1.5 px-2 bg-white hover:bg-slate-50 text-slate-700 text-xs font-semibold rounded-lg border border-slate-200 text-center transition-colors flex items-center justify-center gap-1 shadow-2xs"
                  >
                    <Eye className="w-3.5 h-3.5 text-slate-400 group-hover:text-slate-600" />
                    <span>Просмотр</span>
                  </button>

                  <Link
                    to={`/lots/new?item_id=${item.id}`}
                    className="py-1.5 px-2 bg-white hover:bg-teal-600 hover:text-white text-teal-800 text-xs font-semibold rounded-lg border border-teal-200 text-center transition-colors flex items-center justify-center gap-1 shadow-2xs"
                  >
                    <Gavel className="w-3.5 h-3.5" />
                    <span>Создать лот</span>
                  </Link>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Модалка загрузки нового предмета */}
      {showAddModal && (
        <AddItemForm
          onSuccess={(newItem) => {
            onItemAdded(newItem);
            setShowAddModal(false);
          }}
          onClose={() => setShowAddModal(false)}
        />
      )}

      {/* Модалка просмоторщика предмета */}
      <ItemViewerModal
        item={selectedItemForView}
        onClose={() => setSelectedItemForView(null)}
        onGifted={(id) => {
          onItemRemoved?.(id);
          setSelectedItemForView(null);
        }}
      />
    </div>
  );
}
