import React from 'react';
import { FileBox, HardDrive, CheckCircle2 } from 'lucide-react';
import type { LotDescription } from '~/api/types';

interface GalleryProps {
  description: LotDescription;
  title: string;
  category: string;
}

export function Gallery({ title, category, description }: GalleryProps) {
  const fileSize = description.attributes?.['Размер файла'] || 'Цифровой контент';

  return (
    <div className="flex flex-col gap-2.5">
      {/* Компактный контейнер изображения 16:9 с максимальной высотой */}
      <div className="relative aspect-16/9 max-h-72 w-full bg-slate-100 rounded-xl overflow-hidden border border-slate-200/80 shadow-xs">
        <img
          src="/placeholder.svg"
          alt={title}
          className="w-full h-full object-cover"
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
      </div>
    </div>
  );
}
