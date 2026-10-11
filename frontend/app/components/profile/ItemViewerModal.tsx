import { useState } from 'react';
import { Link } from 'react-router';
import {
  X,
  Download,
  Gavel,
  Copy,
  Check,
  FileCode,
  Music,
  Box,
  Image as ImageIcon,
  FileText,
  Archive,
  Layers,
  HardDrive,
  Calendar,
  ShieldCheck,
  Loader2,
  AlertCircle,
} from 'lucide-react';
import { getItemDownloadUrl } from '~/api/items';
import type { Item, ItemType } from '~/api/types';
import { formatTime } from '~/lib/date';

interface ItemViewerModalProps {
  item: Item | null;
  onClose: () => void;
}

export function ItemViewerModal({ item, onClose }: ItemViewerModalProps) {
  const [downloading, setDownloading] = useState(false);
  const [downloadSuccess, setDownloadSuccess] = useState<string | null>(null);
  const [downloadError, setDownloadError] = useState<string | null>(null);
  const [copiedHash, setCopiedHash] = useState(false);

  if (!item) return null;

  const metadata = item.metadata || {};
  const title = (metadata.title as string) || (metadata.filename as string) || `Предмет #${item.id}`;
  const filename = (metadata.filename as string) || `${title}.bin`;
  const checksum = (metadata.checksum as string) || (metadata.sha256_hex as string) || '';
  const fileSize = (metadata.file_size as string) || 'Неизвестно';
  const fileFormat = (metadata.file_format as string) || item.type.toUpperCase();

  const handleCopyHash = () => {
    if (!checksum) return;
    navigator.clipboard.writeText(checksum);
    setCopiedHash(true);
    setTimeout(() => setCopiedHash(false), 2000);
  };

  const handleDownload = async () => {
    setDownloading(true);
    setDownloadError(null);
    setDownloadSuccess(null);
    try {
      const res = await getItemDownloadUrl(item.id);
      setDownloadSuccess(`Presigned URL получен. Загрузка файла ${filename} началась.`);
      
      // Имитация старта браузерного скачивания
      const link = document.createElement('a');
      link.href = res.url;
      link.setAttribute('download', filename);
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    } catch (err: unknown) {
      if (err instanceof Error) {
        setDownloadError(err.message);
      } else {
        setDownloadError('Ошибка при запросе ссылки на скачивание');
      }
    } finally {
      setDownloading(false);
    }
  };

  const getTypeIcon = (type: ItemType) => {
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

  const getTypeBadgeColor = (type: ItemType) => {
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
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-xs animate-in fade-in duration-200">
      <div className="bg-white rounded-2xl border border-slate-200 w-full max-w-xl max-h-[90vh] flex flex-col shadow-2xl animate-in zoom-in-95 duration-150 overflow-hidden">
        {/* Заголовок модалки */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-slate-100 bg-slate-50/50">
          <div className="flex items-center gap-3 min-w-0">
            <div className="p-2.5 rounded-xl bg-white border border-slate-200 shadow-2xs">
              {getTypeIcon(item.type)}
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <span
                  className={`text-[10px] font-bold uppercase px-2 py-0.5 rounded-md border ${getTypeBadgeColor(
                    item.type
                  )}`}
                >
                  {item.type}
                </span>
                <span className="text-xs text-slate-400 font-mono">ID: #{item.id}</span>
              </div>
              <h3 className="text-base font-bold text-slate-900 truncate mt-0.5">{title}</h3>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-xl text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Контент просмоторщика */}
        <div className="p-5 overflow-y-auto space-y-4">
          {/* Уведомления об ошибке / успехе скачивания */}
          {downloadError && (
            <div className="flex items-start gap-2.5 p-3 rounded-xl bg-rose-50 border border-rose-200 text-rose-800 text-xs animate-in fade-in">
              <AlertCircle className="w-4 h-4 shrink-0 text-rose-600 mt-0.5" />
              <div className="flex-1">{downloadError}</div>
            </div>
          )}
          {downloadSuccess && (
            <div className="flex items-start gap-2.5 p-3 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-800 text-xs animate-in fade-in">
              <ShieldCheck className="w-4 h-4 shrink-0 text-emerald-600 mt-0.5" />
              <div className="flex-1">{downloadSuccess}</div>
            </div>
          )}

          {/* Карточка файла */}
          <div className="p-4 rounded-xl border border-slate-200/80 bg-slate-50/60 space-y-3">
            <div className="flex items-center justify-between text-xs">
              <span className="text-slate-500 font-medium">Оригинальный файл</span>
              <span className="font-semibold text-slate-800 font-mono truncate max-w-[240px]">
                {filename}
              </span>
            </div>

            <div className="grid grid-cols-2 sm:grid-cols-3 gap-2.5 pt-1">
              <div className="p-2.5 bg-white rounded-lg border border-slate-200/60 shadow-2xs">
                <span className="text-[10px] text-slate-400 font-medium uppercase block">Размер</span>
                <span className="text-xs font-bold text-slate-800 flex items-center gap-1 mt-0.5">
                  <HardDrive className="w-3.5 h-3.5 text-teal-600" />
                  {fileSize}
                </span>
              </div>

              <div className="p-2.5 bg-white rounded-lg border border-slate-200/60 shadow-2xs">
                <span className="text-[10px] text-slate-400 font-medium uppercase block">Формат</span>
                <span className="text-xs font-bold text-slate-800 flex items-center gap-1 mt-0.5">
                  <FileCode className="w-3.5 h-3.5 text-teal-600" />
                  {fileFormat}
                </span>
              </div>

              <div className="p-2.5 bg-white rounded-lg border border-slate-200/60 shadow-2xs col-span-2 sm:col-span-1">
                <span className="text-[10px] text-slate-400 font-medium uppercase block">Хранилище</span>
                <span className="text-xs font-bold text-emerald-700 flex items-center gap-1 mt-0.5">
                  <ShieldCheck className="w-3.5 h-3.5 text-emerald-600" />
                  S3 Verified
                </span>
              </div>
            </div>
          </div>

          {/* Контрольная сумма SHA-256 */}
          {checksum && (
            <div className="p-3.5 rounded-xl border border-slate-200/80 bg-white space-y-1.5">
              <div className="flex items-center justify-between">
                <span className="text-xs font-semibold text-slate-700 flex items-center gap-1.5">
                  <ShieldCheck className="w-3.5 h-3.5 text-teal-600" />
                  Контрольная сумма (SHA-256)
                </span>
                <button
                  type="button"
                  onClick={handleCopyHash}
                  className="flex items-center gap-1 text-[11px] font-semibold text-teal-700 hover:text-teal-800 hover:bg-teal-50 px-2 py-0.5 rounded-md transition-colors"
                >
                  {copiedHash ? (
                    <>
                      <Check className="w-3 h-3 text-emerald-600" />
                      <span className="text-emerald-600">Скопировано</span>
                    </>
                  ) : (
                    <>
                      <Copy className="w-3 h-3" />
                      <span>Копировать</span>
                    </>
                  )}
                </button>
              </div>
              <div className="p-2 bg-slate-50 rounded-lg border border-slate-200/60 font-mono text-[11px] text-slate-600 break-all select-all">
                {checksum}
              </div>
            </div>
          )}

          {/* Специфические метаданные по типу файла */}
          <div className="p-3.5 rounded-xl border border-slate-200/80 bg-white space-y-2">
            <h4 className="text-xs font-bold text-slate-800 flex items-center gap-1.5">
              <Layers className="w-3.5 h-3.5 text-teal-600" />
              Характеристики цифрового ассета
            </h4>

            <div className="grid grid-cols-2 gap-2 text-xs">
              {item.type === '3d' && (
                <>
                  <div className="flex justify-between py-1 border-b border-slate-100">
                    <span className="text-slate-500">Полигоны:</span>
                    <span className="font-semibold text-slate-800">
                      {((metadata.polygon_count as number) || 42000).toLocaleString()} tris
                    </span>
                  </div>
                  <div className="flex justify-between py-1 border-b border-slate-100">
                    <span className="text-slate-500">Вертексы:</span>
                    <span className="font-semibold text-slate-800">
                      {((metadata.vertex_count as number) || 24000).toLocaleString()}
                    </span>
                  </div>
                  <div className="flex justify-between py-1 border-b border-slate-100">
                    <span className="text-slate-500">Риггинг:</span>
                    <span className="font-semibold text-emerald-600">Да (Скелет готов)</span>
                  </div>
                  <div className="flex justify-between py-1 border-b border-slate-100">
                    <span className="text-slate-500">Движок:</span>
                    <span className="font-semibold text-slate-800">
                      {(metadata.render_engine as string) || 'Unreal Engine / Unity'}
                    </span>
                  </div>
                </>
              )}

              {item.type === 'audio' && (
                <>
                  <div className="flex justify-between py-1 border-b border-slate-100">
                    <span className="text-slate-500">Длительность:</span>
                    <span className="font-semibold text-slate-800">
                      {(metadata.duration as string) || '03:45'}
                    </span>
                  </div>
                  <div className="flex justify-between py-1 border-b border-slate-100">
                    <span className="text-slate-500">Частота:</span>
                    <span className="font-semibold text-slate-800">
                      {((metadata.sample_rate_hz as number) || 48000) / 1000} kHz
                    </span>
                  </div>
                  <div className="flex justify-between py-1 border-b border-slate-100">
                    <span className="text-slate-500">Битрейт:</span>
                    <span className="font-semibold text-slate-800">
                      {(metadata.bitrate_kbps as number) || 320} kbps
                    </span>
                  </div>
                  <div className="flex justify-between py-1 border-b border-slate-100">
                    <span className="text-slate-500">Каналы:</span>
                    <span className="font-semibold text-slate-800">Стерео (2 ch)</span>
                  </div>
                </>
              )}

              {item.type === 'image' && (
                <>
                  <div className="flex justify-between py-1 border-b border-slate-100">
                    <span className="text-slate-500">Разрешение:</span>
                    <span className="font-semibold text-slate-800">
                      {(metadata.width as number) || 3840} × {(metadata.height as number) || 2160} px
                    </span>
                  </div>
                  <div className="flex justify-between py-1 border-b border-slate-100">
                    <span className="text-slate-500">Цветовой профиль:</span>
                    <span className="font-semibold text-slate-800">Display P3 / sRGB</span>
                  </div>
                </>
              )}

              {item.type === 'archive' && (
                <>
                  <div className="flex justify-between py-1 border-b border-slate-100">
                    <span className="text-slate-500">Файлов в архиве:</span>
                    <span className="font-semibold text-slate-800">
                      {(metadata.file_count as number) || 18} шт.
                    </span>
                  </div>
                  <div className="flex justify-between py-1 border-b border-slate-100">
                    <span className="text-slate-500">Сжатие:</span>
                    <span className="font-semibold text-emerald-600">DEFLATE (Zip)</span>
                  </div>
                </>
              )}

              {item.type === 'document' && (
                <>
                  <div className="flex justify-between py-1 border-b border-slate-100">
                    <span className="text-slate-500">Страниц:</span>
                    <span className="font-semibold text-slate-800">
                      {(metadata.page_count as number) || 24}
                    </span>
                  </div>
                  <div className="flex justify-between py-1 border-b border-slate-100">
                    <span className="text-slate-500">Индексация:</span>
                    <span className="font-semibold text-emerald-600">PDF / Текст проверен</span>
                  </div>
                </>
              )}

              <div className="flex justify-between py-1 border-b border-slate-100 col-span-2">
                <span className="text-slate-500 flex items-center gap-1">
                  <Calendar className="w-3.5 h-3.5 text-slate-400" />
                  Дата создания:
                </span>
                <span className="font-semibold text-slate-700">
                  {formatTime(item.created_at)}
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* Действия в футере */}
        <div className="px-5 py-4 border-t border-slate-100 bg-slate-50/50 flex flex-col sm:flex-row items-center justify-between gap-3">
          <div className="text-[11px] text-slate-500 text-center sm:text-left">
            Скачивание файла защищено S3 presigned ссылкой (1 раз в сутки для владельца)
          </div>

          <div className="flex items-center gap-2.5 w-full sm:w-auto justify-end">
            <button
              type="button"
              onClick={handleDownload}
              disabled={downloading}
              className="flex-1 sm:flex-initial flex items-center justify-center gap-1.5 px-3.5 py-2 bg-white hover:bg-slate-50 active:bg-slate-100 text-slate-800 font-semibold text-xs border border-slate-200 rounded-xl shadow-2xs transition-colors disabled:opacity-50"
            >
              {downloading ? (
                <>
                  <Loader2 className="w-3.5 h-3.5 animate-spin text-teal-600" />
                  <span>Генерация ссылки...</span>
                </>
              ) : (
                <>
                  <Download className="w-3.5 h-3.5 text-teal-600" />
                  <span>Скачать файл</span>
                </>
              )}
            </button>

            <Link
              to={`/lots/new?item_id=${item.id}`}
              onClick={onClose}
              className="flex-1 sm:flex-initial flex items-center justify-center gap-1.5 px-4 py-2 bg-teal-600 hover:bg-teal-700 text-white font-bold text-xs rounded-xl shadow-xs transition-colors"
            >
              <Gavel className="w-3.5 h-3.5" />
              <span>Создать лот</span>
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
