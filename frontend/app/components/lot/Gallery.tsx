import React, { useState } from 'react';
import { FileBox, HardDrive, Images } from 'lucide-react';
import type { LotDescription } from '~/api/types';
import { resolvePhotoUrl } from '~/api/lots';

interface GalleryProps {
  description: LotDescription;
  title: string;
  category: string;
}

export function Gallery({ title, category, description }: GalleryProps) {
  const fileSize = description.attributes?.['Размер файла'] || 'Цифровой контент';

  const photos: string[] = [];
  const rawList = Array.isArray(description.images) && description.images.length > 0
    ? description.images
    : Array.isArray(description.photo)
    ? description.photo
    : [];

  for (const p of rawList) {
    if (typeof p === 'string' && p.trim() !== '') {
      const resolved = resolvePhotoUrl(p);
      if (!photos.includes(resolved)) {
        photos.push(resolved);
      }
    }
  }

  if (Array.isArray(description.media)) {
    for (const m of description.media) {
      const u = (m as any).url || m.id;
      if (typeof u === 'string' && u.trim() !== '') {
        const resolved = resolvePhotoUrl(u);
        if (!photos.includes(resolved)) {
          photos.push(resolved);
        }
      }
    }
  }

  const [selectedIdx, setSelectedIdx] = useState(0);
  const activePhoto = photos[selectedIdx] || '/placeholder.svg';

  return (
    <div className="flex flex-col gap-2.5">
      {/* Главный контейнер изображения 16:9 с максимальной высотой */}
      <div className="relative aspect-16/9 max-h-80 w-full bg-slate-100 rounded-2xl overflow-hidden border border-slate-200/80 shadow-xs">
        <img
          src={activePhoto}
          alt={title}
          onError={(e) => {
            (e.currentTarget as HTMLImageElement).src = '/placeholder.svg';
          }}
          className="w-full h-full object-cover transition-all duration-300"
        />

        {/* Плашка цифрового файла */}
        <div className="absolute bottom-2.5 left-2.5 flex items-center gap-2">
          <span className="px-2.5 py-1 rounded-lg bg-slate-900/85 backdrop-blur-xs text-white text-[11px] font-semibold flex items-center gap-1.5 shadow-sm">
            <FileBox className="w-3.5 h-3.5 text-teal-400" />
            <span>{category}</span>
          </span>
          <span className="px-2 py-1 rounded-lg bg-white/90 backdrop-blur-xs text-slate-800 text-[11px] font-semibold flex items-center gap-1 shadow-sm">
            <HardDrive className="w-3 h-3 text-slate-500" />
            <span>{fileSize}</span>
          </span>
        </div>

        {photos.length > 1 && (
          <div className="absolute top-2.5 right-2.5 px-2 py-1 rounded-lg bg-slate-900/75 backdrop-blur-xs text-white text-[11px] font-semibold flex items-center gap-1.5 shadow-sm">
            <Images className="w-3 h-3 text-teal-300" />
            <span>{selectedIdx + 1} / {photos.length}</span>
          </div>
        )}
      </div>

      {/* Миниатюры фотографий при наличии нескольких фото */}
      {photos.length > 1 && (
        <div className="flex items-center gap-2 overflow-x-auto pb-1">
          {photos.map((p, idx) => (
            <button
              key={idx}
              type="button"
              onClick={() => setSelectedIdx(idx)}
              className={`relative w-16 h-12 rounded-xl overflow-hidden border-2 transition-all shrink-0 cursor-pointer ${
                selectedIdx === idx
                  ? 'border-teal-600 ring-2 ring-teal-500/30'
                  : 'border-slate-200 hover:border-slate-300 opacity-70 hover:opacity-100'
              }`}
            >
              <img
                src={p}
                alt={`${title} #${idx + 1}`}
                onError={(e) => {
                  (e.currentTarget as HTMLImageElement).src = '/placeholder.svg';
                }}
                className="w-full h-full object-cover"
              />
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
