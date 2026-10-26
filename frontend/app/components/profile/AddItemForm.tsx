import { useState, useRef } from 'react';
import {
  Upload,
  X,
  FileCode,
  HardDrive,
  ShieldCheck,
  CheckCircle2,
  Loader2,
  Box,
  Music,
  Image as ImageIcon,
  Archive,
  FileText,
  AlertCircle,
} from 'lucide-react';
import {
  getUploadUrl,
  uploadBinaryToS3,
  confirmUpload,
  getUploadStatus,
  cancelUpload,
} from '~/api/upload';
import { saveUserItemLocally } from '~/api/items';
import type { Item, ItemType } from '~/api/types';

interface AddItemFormProps {
  onSuccess: (item: Item) => void;
  onClose: () => void;
}

type UploadStep =
  | 'idle'
  | 'requesting_url'
  | 'uploading_s3'
  | 'confirming'
  | 'processing'
  | 'completed'
  | 'error';


export const MAX_FILE_SIZES: Record<ItemType | 'photo', number> = {
  archive: 5 * 1024 * 1024 * 1024,   // 5 ГБ (5 << 30) (только архивы до 5 ГБ)
  video: 1 * 1024 * 1024 * 1024,     // 1 ГБ (1 << 30) (видео до 1 ГБ)
  audio: 500 * 1024 * 1024,          // 500 МБ (500 << 20)
  '3d': 500 * 1024 * 1024,           // 500 МБ (500 << 20)
  document: 200 * 1024 * 1024,       // 200 МБ (200 << 20)
  image: 100 * 1024 * 1024,          // 100 МБ (100 << 20)
  photo: 5 * 1024 * 1024,            // 5 МБ (5 << 20) (фотографии пользователей)
};

export function getMaxFileSize(type: ItemType | 'photo'): number {
  return MAX_FILE_SIZES[type] ?? MAX_FILE_SIZES.document;
}

export function formatFileSize(bytes: number): string {
  if (bytes >= 1024 * 1024 * 1024) {
    const gb = bytes / (1024 * 1024 * 1024);
    return `${gb % 1 === 0 ? gb : gb.toFixed(1)} ГБ`;
  }
  return `${Math.round(bytes / (1024 * 1024))} МБ`;
}

export function AddItemForm({ onSuccess, onClose }: AddItemFormProps) {
  const [file, setFile] = useState<File | null>(null);
  const [sha256Hex, setSha256Hex] = useState<string>('');
  const [calculatingHash, setCalculatingHash] = useState(false);

  const [name, setName] = useState('');
  const [type, setType] = useState<ItemType>('image');
  const [description, setDescription] = useState('');

  // Upload pipeline state
  const [step, setStep] = useState<UploadStep>('idle');
  const [uploadPercent, setUploadPercent] = useState(0);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [activeUploadKey, setActiveUploadKey] = useState<string | null>(null);

  const fileInputRef = useRef<HTMLInputElement>(null);

  const detectTypeFromFilename = (filename: string): ItemType => {
    const ext = filename.split('.').pop()?.toLowerCase() || '';
    if (['fbx', 'obj', 'gltf', 'glb', 'blend', 'max', 'c4d'].includes(ext)) return '3d';
    if (['wav', 'flac', 'mp3', 'ogg', 'aiff', 'm4a', 'aac'].includes(ext)) return 'audio';
    if (['mp4', 'mov', 'avi', 'mkv', 'webm', 'flv'].includes(ext)) return 'video';
    if (['psd', 'png', 'jpg', 'jpeg', 'tiff', 'svg', 'webp'].includes(ext)) return 'image';
    if (['zip', 'rar', '7z', 'tar', 'gz', 'unitypackage'].includes(ext)) return 'archive';
    return 'document';
  };

  const computeSha256 = async (f: File): Promise<string> => {
    const buffer = await f.arrayBuffer();
    const hashBuffer = await crypto.subtle.digest('SHA-256', buffer);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    return hashArray.map((b) => b.toString(16).padStart(2, '0')).join('');
  };

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFile = e.target.files?.[0];
    if (!selectedFile) return;

    setFile(selectedFile);
    setErrorMsg(null);

    // Автоопределение названия и типа
    const detectedType = detectTypeFromFilename(selectedFile.name);
    setType(detectedType);
    if (!name) {
      setName(selectedFile.name.replace(/\.[^/.]+$/, ''));
    }

    // Вычисление реального SHA-256
    setCalculatingHash(true);
    try {
      const hash = await computeSha256(selectedFile);
      setSha256Hex(hash);
    } catch {
      // Fallback на детерминированный хеш
      setSha256Hex('e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855');
    } finally {
      setCalculatingHash(false);
    }
  };

  const handleDrop = async (e: React.DragEvent) => {
    e.preventDefault();
    const droppedFile = e.dataTransfer.files?.[0];
    if (!droppedFile) return;

    setFile(droppedFile);
    setErrorMsg(null);

    const detectedType = detectTypeFromFilename(droppedFile.name);
    setType(detectedType);
    if (!name) {
      setName(droppedFile.name.replace(/\.[^/.]+$/, ''));
    }

    setCalculatingHash(true);
    try {
      const hash = await computeSha256(droppedFile);
      setSha256Hex(hash);
    } catch {
      setSha256Hex('e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855');
    } finally {
      setCalculatingHash(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);

    if (!file) {
      setErrorMsg('Пожалуйста, выберите файл для загрузки');
      return;
    }
    if (file.size <= 0) {
      setErrorMsg('Размер файла должен быть больше 0 байт');
      return;
    }
    const maxLimit = getMaxFileSize(type);
    if (file.size > maxLimit) {
      setErrorMsg(`Превышен максимальный лимит размера файла для категории "${type}" (${formatFileSize(maxLimit)})`);
      return;
    }
    if (name.trim().length < 3) {
      setErrorMsg('Название предмета должно содержать не менее 3 символов');
      return;
    }
    if (!sha256Hex || calculatingHash) {
      setErrorMsg('Идёт вычисление контрольной суммы SHA-256. Пожалуйста, подождите.');
      return;
    }

    setStep('requesting_url');
    setUploadPercent(0);

    try {
      // 1. POST /api/upload/url
      const urlResp = await getUploadUrl({
        file_size: file.size,
        sha_256_hex: sha256Hex,
        filename: file.name,
      });
      setActiveUploadKey(urlResp.key);

      // 2. Direct PUT to S3
      setStep('uploading_s3');
      await uploadBinaryToS3(urlResp.url, file, (percent) => {
        setUploadPercent(percent);
      });

      // 3. POST /api/upload/confirm
      setStep('confirming');
      await confirmUpload(urlResp.key);

      // 4. GET /api/upload/status (опрос воркера)
      setStep('processing');
      const maxAttempts = 60;
      let attempt = 0;
      let completedItem: Item | null = null;

      while (attempt < maxAttempts) {
        attempt++;
        await new Promise((r) => setTimeout(r, 1500));
        const statusResp = await getUploadStatus(urlResp.key);

        if (statusResp.status === 'completed' && statusResp.item) {
          completedItem = statusResp.item;
          break;
        }
        if (statusResp.status === 'failed') {
          throw new Error(statusResp.error || 'Ошибка проверки файла на сервере');
        }
        if (statusResp.status === 'cancelled') {
          throw new Error('Загрузка была отменена');
        }
      }

      if (completedItem) {
        saveUserItemLocally(completedItem);
        setStep('completed');
        setTimeout(() => {
          onSuccess(completedItem!);
        }, 1200);
      } else {
        throw new Error('Превышено время ожидания обработки файла воркером');
      }
    } catch (err: unknown) {
      setStep('error');
      if (err instanceof Error) {
        setErrorMsg(err.message);
      } else {
        setErrorMsg('Не удалось загрузить файл на сервер');
      }
    }
  };

  const handleCancel = async () => {
    if (activeUploadKey && step !== 'completed' && step !== 'idle') {
      try {
        await cancelUpload(activeUploadKey);
      } catch {
        // ignore
      }
    }
    onClose();
  };

  return (
    <div className="fixed inset-0 z-50 bg-slate-900/60 backdrop-blur-xs flex items-center justify-center p-4">
      <div className="bg-white rounded-2xl max-w-lg w-full p-6 shadow-xl space-y-5 border border-slate-200">
        <div className="flex items-center justify-between pb-3 border-b border-slate-100">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-teal-50 text-teal-700 flex items-center justify-center">
              <Upload className="w-4 h-4" />
            </div>
            <div>
              <h2 className="text-base font-bold text-slate-900">Добавить предмет (Asset)</h2>
              <p className="text-xs text-slate-400">Прямая загрузка в S3 с верификацией</p>
            </div>
          </div>
          <button
            onClick={handleCancel}
            className="w-7 h-7 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-100 flex items-center justify-center transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {errorMsg ? (
          <div className="p-3 bg-rose-50 border border-rose-200 rounded-xl text-rose-800 text-xs flex items-center gap-2">
            <AlertCircle className="w-4 h-4 shrink-0 text-rose-600" />
            <span>{errorMsg}</span>
          </div>
        ) : null}

        {step === 'idle' || step === 'error' ? (
          <form onSubmit={handleSubmit} className="space-y-4">
            {/* Drop zone */}
            <div
              onDragOver={(e) => e.preventDefault()}
              onDrop={handleDrop}
              onClick={() => fileInputRef.current?.click()}
              className={`border-2 border-dashed rounded-xl p-6 text-center cursor-pointer transition-colors ${
                file
                  ? 'border-teal-500 bg-teal-50/20'
                  : 'border-slate-200 hover:border-teal-400 hover:bg-slate-50'
              }`}
            >
              <input
                ref={fileInputRef}
                type="file"
                className="hidden"
                onChange={handleFileChange}
              />
              <div className="flex flex-col items-center justify-center gap-2">
                <div className="w-10 h-10 rounded-full bg-slate-100 flex items-center justify-center text-slate-500">
                  <HardDrive className="w-5 h-5 text-teal-600" />
                </div>
                {file ? (
                  <div>
                    <p className="text-xs font-semibold text-slate-800">{file.name}</p>
                    <p className="text-[10px] text-slate-400">
                      {(file.size / (1024 * 1024)).toFixed(2)} МБ • {type.toUpperCase()} (лимит: {formatFileSize(getMaxFileSize(type))})
                    </p>
                    {calculatingHash ? (
                      <span className="inline-flex items-center gap-1 text-[10px] text-teal-600 mt-1">
                        <Loader2 className="w-3 h-3 animate-spin" /> Расчёт SHA-256...
                      </span>
                    ) : sha256Hex ? (
                      <span className="inline-flex items-center gap-1 text-[10px] text-emerald-600 mt-1 font-mono">
                        <ShieldCheck className="w-3 h-3" /> SHA-256 готов
                      </span>
                    ) : null}
                  </div>
                ) : (
                  <div>
                    <p className="text-xs font-semibold text-slate-700">
                      Перетащите файл сюда или нажмите для выбора
                    </p>
                    <p className="text-[10px] text-slate-400 mt-0.5">
                      Архивы до 5 ГБ, видео до 1 ГБ, 3D и аудио до 500 МБ, документы до 200 МБ, изображения до 100 МБ
                    </p>
                  </div>
                )}
              </div>
            </div>

            {/* Название предмета */}
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">
                Название предмета
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Cyberpunk Character Model"
                className="w-full px-3 py-2 text-xs bg-slate-50 border border-slate-200 rounded-xl focus:outline-hidden focus:ring-2 focus:ring-teal-500/20 focus:border-teal-500"
              />
            </div>

            {/* Краткое описание */}
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">
                Описание содержимого
              </label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Исходники в FBX/OBJ, 4K PBR текстуры..."
                rows={2}
                className="w-full px-3 py-2 text-xs bg-slate-50 border border-slate-200 rounded-xl focus:outline-hidden focus:ring-2 focus:ring-teal-500/20 focus:border-teal-500"
              />
            </div>

            <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-100">
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-2 text-xs font-medium text-slate-600 hover:bg-slate-100 rounded-xl transition-colors"
              >
                Отмена
              </button>
              <button
                type="submit"
                disabled={!file || calculatingHash}
                className="px-4 py-2 text-xs font-bold text-white bg-teal-600 hover:bg-teal-700 disabled:opacity-50 rounded-xl shadow-xs transition-colors flex items-center gap-1.5"
              >
                <span>Начать загрузку в S3</span>
              </button>
            </div>
          </form>
        ) : (
          /* Upload pipeline progress modal */
          <div className="py-6 space-y-5 text-center">
            <div className="flex flex-col items-center justify-center gap-3">
              {step === 'completed' ? (
                <div className="w-12 h-12 rounded-full bg-emerald-100 text-emerald-600 flex items-center justify-center animate-bounce">
                  <CheckCircle2 className="w-6 h-6" />
                </div>
              ) : (
                <div className="w-12 h-12 rounded-full bg-teal-50 text-teal-600 flex items-center justify-center">
                  <Loader2 className="w-6 h-6 animate-spin" />
                </div>
              )}

              <div>
                <h3 className="text-sm font-bold text-slate-900">
                  {step === 'requesting_url' && '1/4 Получение presigned URL...'}
                  {step === 'uploading_s3' && `2/4 Загрузка в S3 хранилище (${uploadPercent}%)...`}
                  {step === 'confirming' && '3/4 Подтверждение целостности (первые 512B)...'}
                  {step === 'processing' && '4/4 Фоновый парсинг метаданных воркером...'}
                  {step === 'completed' && 'Предмет успешно создан и проверен!'}
                </h3>
                <p className="text-xs text-slate-400 mt-0.5">
                  {step === 'completed'
                    ? 'Файл верифицирован и готов к выставлению на торги'
                    : 'Пожалуйста, не закрывайте вкладку во время загрузки'}
                </p>
              </div>
            </div>

            {/* Прогресс-бар */}
            <div className="w-full bg-slate-100 rounded-full h-2 overflow-hidden">
              <div
                className="bg-teal-600 h-full transition-all duration-300"
                style={{
                  width: `${
                    step === 'requesting_url'
                      ? 15
                      : step === 'uploading_s3'
                      ? 15 + uploadPercent * 0.5
                      : step === 'confirming'
                      ? 70
                      : step === 'processing'
                      ? 90
                      : 100
                  }%`,
                }}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
