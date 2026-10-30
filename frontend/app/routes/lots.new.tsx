import { useState, useEffect, useRef, useCallback } from 'react';
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
  Upload,
  X,
  ImageIcon,
  GripVertical,
  Star,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react';
import { useAuth } from '~/hooks/useAuth';
import { listUserItems } from '~/api/items';
import { createLot, getMyLots, updateLot, activateLot, getLot, cacheLot, uploadLotPhoto, resolvePhotoUrl } from '~/api/lots';
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

const DRAFT_STORAGE_KEY_PREFIX = 'auc_lot_draft_form_';

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
  const [photos, setPhotos] = useState<string[]>([]);
  const [uploadingPhoto, setUploadingPhoto] = useState(false);
  const [photoError, setPhotoError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Состояние Drag-and-Drop карточек фотографий
  const [draggedPhotoIdx, setDraggedPhotoIdx] = useState<number | null>(null);
  const [dragOverPhotoIdx, setDragOverPhotoIdx] = useState<number | null>(null);

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

  // Состояние автосохранения черновика
  const [draftSaving, setDraftSaving] = useState(false);
  const [draftSavedAt, setDraftSavedAt] = useState<Date | null>(null);
  const [restoredFromLocal, setRestoredFromLocal] = useState(false);
  const lastServerSaveTimeRef = useRef<number>(0);
  const isDirtyRef = useRef(false);
  const editingLotIdRef = useRef<number | null>(editingLotId);
  editingLotIdRef.current = editingLotId;
  const draftLoadedRef = useRef(false);

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

    const rawPhotos = lot.description?.images || lot.description?.photo || [];
    const lotPhotos = Array.isArray(rawPhotos)
      ? rawPhotos
          .filter((p: any) => typeof p === 'string' && p.trim() !== '')
          .map((p: string) => p.replace(/^\/api\/lot\/photo\//, '').replace(/^.*\//, '').trim())
          .filter(Boolean)
      : (lot.description?.media || [])
          .map((m: any) => typeof m?.url === 'string' ? m.url.replace(/^\/api\/lot\/photo\//, '').replace(/^.*\//, '').trim() : '')
          .filter(Boolean);
    setPhotos(lotPhotos);

    if (lot.started_at) {
      setStartDateStr(toLocalISOString(new Date(lot.started_at)));
    }
    if (lot.ends_at) {
      setEndDateStr(toLocalISOString(new Date(lot.ends_at)));
    }

    if (lot.description?.draft_ui) {
      try {
        const ui = JSON.parse(lot.description.draft_ui);
        if (ui.quickDuration) setQuickDuration(ui.quickDuration);
        if (ui.autoActivate !== undefined) setAutoActivate(ui.autoActivate);
        if (ui.isDraft !== undefined) setIsDraft(ui.isDraft);
      } catch {
        // ignore
      }
    }
  };

  const resetToCreateNew = () => {
    setEditingLotId(null);
    setEditingLotStatus(null);
    setSearchParams({});
    setRestoredFromLocal(false);
    if (user?.id) {
      try {
        localStorage.removeItem(`${DRAFT_STORAGE_KEY_PREFIX}${user.id}`);
      } catch {}
    }
    setTitle('');
    setPhotos([]);
    setPhotoError(null);
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

  const handlePhotoFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files || files.length === 0) return;

    if (photos.length >= 10) {
      setPhotoError('Максимальное количество фотографий для описания лота — 10');
      return;
    }

    setUploadingPhoto(true);
    setPhotoError(null);
    try {
      const remainingSlots = 10 - photos.length;
      const filesToUpload = Array.from(files).slice(0, remainingSlots);

      for (const file of filesToUpload) {
        if (file.size > 5 * 1024 * 1024) {
          throw new Error(`Файл "${file.name}" превышает допустимый размер 5 МБ`);
        }
        const res = await uploadLotPhoto(file);
        const photoUUID = (res.id || '').replace(/^\/api\/lot\/photo\//, '').replace(/^.*\//, '').trim();
        setPhotos((prev) => [...prev, photoUUID || res.url]);
      }
    } catch (err: unknown) {
      setPhotoError(err instanceof Error ? err.message : 'Не удалось загрузить фотографию');
    } finally {
      setUploadingPhoto(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  const handleRemovePhoto = (idxToRemove: number) => {
    setPhotos((prev) => prev.filter((_, idx) => idx !== idxToRemove));
  };

  // Drag-and-drop перетаскивание карточек фотографий
  const handlePhotoDragStart = (e: React.DragEvent, idx: number) => {
    setDraggedPhotoIdx(idx);
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', String(idx));
  };

  const handlePhotoDragOver = (e: React.DragEvent, idx: number) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    if (dragOverPhotoIdx !== idx) {
      setDragOverPhotoIdx(idx);
    }
  };

  const handlePhotoDrop = (e: React.DragEvent, targetIdx: number) => {
    e.preventDefault();
    if (draggedPhotoIdx === null || draggedPhotoIdx === targetIdx) {
      setDraggedPhotoIdx(null);
      setDragOverPhotoIdx(null);
      return;
    }
    setPhotos((prev) => {
      const updated = [...prev];
      const [moved] = updated.splice(draggedPhotoIdx, 1);
      updated.splice(targetIdx, 0, moved);
      return updated;
    });
    setDraggedPhotoIdx(null);
    setDragOverPhotoIdx(null);
  };

  const handlePhotoDragEnd = () => {
    setDraggedPhotoIdx(null);
    setDragOverPhotoIdx(null);
  };

  // Перемещение кнопками влево/вправо
  const handleMovePhoto = (fromIdx: number, toIdx: number) => {
    if (toIdx < 0 || toIdx >= photos.length) return;
    setPhotos((prev) => {
      const updated = [...prev];
      const [moved] = updated.splice(fromIdx, 1);
      updated.splice(toIdx, 0, moved);
      return updated;
    });
  };

  // Сделать фото обложкой (позиция №1)
  const handleMakeCoverPhoto = (idx: number) => {
    if (idx === 0) return;
    setPhotos((prev) => {
      const updated = [...prev];
      const [moved] = updated.splice(idx, 1);
      updated.unshift(moved);
      return updated;
    });
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

  // Автоматическое сохранение черновика на сервер
  const saveDraftToServer = useCallback(async () => {
    if (!isAuthenticated || !selectedItemId) return;
    const now = Date.now();
    // Не чаще 1 раза в 5 секунд при переключении и автосохранении
    if (now - lastServerSaveTimeRef.current < 5000) {
      return;
    }

    setDraftSaving(true);
    try {
      const startIso = new Date(startDateStr).toISOString();
      const endIso = new Date(endDateStr).toISOString();
      const durationMin = Math.round(
        (new Date(endDateStr).getTime() - new Date(startDateStr).getTime()) / 60000
      );

      const draftPayload = {
        item_id: selectedItemId,
        title: title.trim() || 'Черновик лота',
        category,
        short_text: shortText.trim(),
        text: text.trim() || shortText.trim() || title.trim() || 'Черновик',
        start_amount: Math.max(100, Math.round(startAmount / 100) * 100),
        min_step: Math.min(500, Math.max(50, Math.round(minStep / 50) * 50)),
        duration_minutes: durationMin > 0 ? durationMin : 1440,
        started_at: startIso,
        ends_at: endIso,
        status: 'draft' as const,
        auto_activate: autoActivate,
        photos: photos,
        images: photos,
        draft_ui: JSON.stringify({
          quickDuration,
          autoActivate,
          isDraft,
        }),
      };

      if (editingLotIdRef.current) {
        const updated = await updateLot(editingLotIdRef.current, draftPayload);
        cacheLot(updated);
      } else {
        const created = await createLot(draftPayload);
        cacheLot(created);
        setEditingLotId(created.id);
        editingLotIdRef.current = created.id;
        setEditingLotStatus('draft');
        if (typeof window !== 'undefined') {
          const url = new URL(window.location.href);
          url.searchParams.set('edit_id', String(created.id));
          window.history.replaceState({}, '', url.toString());
        }
      }

      lastServerSaveTimeRef.current = Date.now();
      isDirtyRef.current = false;
      setDraftSavedAt(new Date());
    } catch (err) {
      console.warn('Auto-save draft error:', err);
    } finally {
      setDraftSaving(false);
    }
  }, [
    isAuthenticated,
    selectedItemId,
    title,
    category,
    shortText,
    text,
    startAmount,
    minStep,
    startDateStr,
    endDateStr,
    autoActivate,
    photos,
    quickDuration,
    isDraft,
  ]);

  const saveDraftRef = useRef(saveDraftToServer);
  saveDraftRef.current = saveDraftToServer;

  // Восстановление черновика из localStorage при открытии без edit_id
  useEffect(() => {
    if (!isAuthenticated || !user || draftLoadedRef.current) return;
    if (editIdParam) return;

    try {
      const raw = localStorage.getItem(`${DRAFT_STORAGE_KEY_PREFIX}${user.id}`);
      if (raw) {
        const saved = JSON.parse(raw);
        if (saved && typeof saved === 'object' && (saved.title || saved.selectedItemId)) {
          if (saved.editingLotId) setEditingLotId(saved.editingLotId);
          if (saved.editingLotStatus) setEditingLotStatus(saved.editingLotStatus);
          if (saved.selectedItemId) setSelectedItemId(saved.selectedItemId);
          if (saved.title !== undefined) setTitle(saved.title);
          if (saved.shortText !== undefined) setShortText(saved.shortText);
          if (saved.text !== undefined) setText(saved.text);
          if (saved.startAmount) setStartAmount(saved.startAmount);
          if (saved.minStep) setMinStep(saved.minStep);
          if (saved.isDraft !== undefined) setIsDraft(saved.isDraft);
          if (saved.autoActivate !== undefined) setAutoActivate(saved.autoActivate);
          if (saved.quickDuration) setQuickDuration(saved.quickDuration);
          if (saved.startDateStr) setStartDateStr(saved.startDateStr);
          if (saved.endDateStr) setEndDateStr(saved.endDateStr);
                    const rawSaved = saved.images || saved.photos;
          if (Array.isArray(rawSaved)) {
            setPhotos(
              rawSaved
                .filter((p: any) => typeof p === 'string' && p.trim() !== '')
                .map((p: string) => p.replace(/^\/api\/lot\/photo\//, '').replace(/^.*\//, '').trim())
                .filter(Boolean)
            );
          }
          setRestoredFromLocal(true);
        }
      }
    } catch {
      // ignore
    }
    draftLoadedRef.current = true;
  }, [isAuthenticated, user, editIdParam]);

  // Сохранение в localStorage при каждом изменении (чтобы не пропадал при переключении URL)
  useEffect(() => {
    if (!isAuthenticated || !user) return;
    if (!draftLoadedRef.current && !editIdParam) return;
    isDirtyRef.current = true;
    try {
      const data = {
        editingLotId,
        editingLotStatus,
        selectedItemId,
        title,
        shortText,
        text,
        startAmount,
        minStep,
        isDraft,
        autoActivate,
        quickDuration,
        startDateStr,
        endDateStr,
        photos,
        images: photos,
        updatedAt: Date.now(),
      };
      localStorage.setItem(`${DRAFT_STORAGE_KEY_PREFIX}${user.id}`, JSON.stringify(data));
    } catch {
      // ignore storage quota
    }
  }, [
    isAuthenticated,
    user,
    editingLotId,
    editingLotStatus,
    selectedItemId,
    title,
    shortText,
    text,
    startAmount,
    minStep,
    isDraft,
    autoActivate,
    quickDuration,
    startDateStr,
    endDateStr,
    photos,
    editIdParam,
  ]);

  // Автосохранение черновика на сервере: подождать 7 секунд после изменения
  useEffect(() => {
    if (!isAuthenticated || !selectedItemId) return;
    if (!isDirtyRef.current) return;

    const timer = setTimeout(() => {
      saveDraftRef.current();
    }, 7000);

    return () => clearTimeout(timer);
  }, [
    selectedItemId,
    title,
    shortText,
    text,
    startAmount,
    minStep,
    startDateStr,
    endDateStr,
    autoActivate,
    photos,
    quickDuration,
    isDraft,
    isAuthenticated,
  ]);

  // При переходе на другой URL (размонтировании) обновлять лот со статусом draft (не чаще 1 раза в 5 секунд)
  useEffect(() => {
    return () => {
      if (isDirtyRef.current && Date.now() - lastServerSaveTimeRef.current >= 5000) {
        saveDraftRef.current();
      }
    };
  }, []);

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
        photos: photos,
        images: photos,
        draft_ui: JSON.stringify({
          quickDuration,
          autoActivate,
          isDraft,
        }),
      };

      if (user?.id) {
        try {
          localStorage.removeItem(`${DRAFT_STORAGE_KEY_PREFIX}${user.id}`);
        } catch {}
      }
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

  // Предпросмотр карточки лота (первое фото всегда главная обложка)
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
    item_summary: {
      id: selectedItemId || 1,
      type: category,
      preview: photos[0] || '/placeholder.svg',
    },
    status: isDraft ? 'draft' : 'active',
    category,
    title: title || 'Название лота',
    description: {
      text: text || 'Полное описание лота...',
      short_text: shortText || 'Краткое описание для карточки',
      photo: photos,
      media: photos.map((p, idx) => ({ id: `m-${idx}`, url: p, type: 'image' as const, order: idx + 1 })),
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
          <div className="flex items-center gap-2.5">
            <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-slate-900">
              {editingLotId ? `Редактировать лот #${editingLotId}` : 'Выставить лот на аукцион'}
            </h1>
            {draftSaving ? (
              <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full bg-teal-50 text-teal-700 text-[10px] font-semibold border border-teal-200 animate-pulse">
                <Loader2 className="w-2.5 h-2.5 animate-spin" />
                <span>Сохранение черновика...</span>
              </span>
            ) : draftSavedAt ? (
              <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full bg-slate-50 text-slate-600 text-[10px] font-medium border border-slate-200">
                <CheckCircle2 className="w-2.5 h-2.5 text-emerald-600" />
                <span>Черновик сохранен ({draftSavedAt.toLocaleTimeString()})</span>
              </span>
            ) : null}
          </div>
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

      {restoredFromLocal && (
        <div className="p-3 bg-sky-50 border border-sky-200 rounded-xl text-sky-900 text-xs flex items-center justify-between gap-2 animate-in fade-in">
          <span>Восстановлен черновик лота из предыдущей сессии. Вы можете продолжить заполнение или сбросить форму.</span>
          <button
            type="button"
            onClick={resetToCreateNew}
            className="text-[11px] font-bold text-sky-700 hover:text-sky-900 underline cursor-pointer shrink-0"
          >
            Сбросить
          </button>
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

          {/* Сетка фотографий лота (до 10 шт.) с Drag & Drop слева направо */}
          <div className="p-4 bg-slate-50 border border-slate-200 rounded-2xl space-y-3 transition-all duration-300">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pb-1 border-b border-slate-200/60">
              <div>
                <label className="text-xs font-bold text-slate-800 flex items-center gap-1.5">
                  <ImageIcon className="w-4 h-4 text-teal-600" />
                  <span>Фотографии лота ({photos.length} из 10)</span>
                </label>
                <p className="text-[11px] text-slate-500 mt-0.5">
                  Сетка упорядочивания слева направо. Перетаскивайте карточки для изменения порядка (№1 — главная обложка).
                </p>
              </div>

              <div className="flex items-center gap-2">
                {photos.length < 10 && (
                  <button
                    type="button"
                    onClick={() => fileInputRef.current?.click()}
                    disabled={uploadingPhoto}
                    className="inline-flex items-center gap-1 px-3 py-1 bg-white hover:bg-teal-50 border border-teal-300 text-teal-700 text-xs font-bold rounded-lg shadow-2xs transition-colors cursor-pointer disabled:opacity-50"
                  >
                    {uploadingPhoto ? (
                      <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    ) : (
                      <Upload className="w-3.5 h-3.5" />
                    )}
                    <span>{uploadingPhoto ? 'Загрузка...' : 'Добавить фото'}</span>
                  </button>
                )}
              </div>
            </div>

            <input
              ref={fileInputRef}
              type="file"
              accept="image/jpeg,image/png,image/webp"
              multiple
              onChange={handlePhotoFileChange}
              className="hidden"
            />

            {photoError && (
              <div className="p-2.5 bg-rose-50 border border-rose-200 rounded-xl text-rose-700 text-xs flex items-center gap-2">
                <AlertCircle className="w-4 h-4 shrink-0 text-rose-600" />
                <span>{photoError}</span>
              </div>
            )}

            {photos.length > 0 ? (
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-3.5 pt-1">
                {photos.map((p, idx) => {
                  const isDragging = draggedPhotoIdx === idx;
                  const isOver = dragOverPhotoIdx === idx && draggedPhotoIdx !== idx;

                  return (
                    <div
                      key={p + idx}
                      draggable
                      onDragStart={(e) => handlePhotoDragStart(e, idx)}
                      onDragOver={(e) => handlePhotoDragOver(e, idx)}
                      onDrop={(e) => handlePhotoDrop(e, idx)}
                      onDragEnd={handlePhotoDragEnd}
                      className={`relative group rounded-xl overflow-hidden border bg-white shadow-2xs aspect-4/3 cursor-grab active:cursor-grabbing select-none transition-all duration-200 ${
                        isDragging
                          ? 'opacity-40 scale-95 border-teal-500 border-dashed ring-2 ring-teal-400'
                          : isOver
                          ? 'border-teal-500 ring-2 ring-teal-500 scale-102 shadow-md'
                          : 'border-slate-200 hover:border-teal-400 hover:shadow-sm'
                      }`}
                    >
                      <img
                        src={p}
                        alt={`Фото ${idx + 1}`}
                        onError={(e) => {
                          (e.currentTarget as HTMLImageElement).src = '/placeholder.svg';
                        }}
                        className="w-full h-full object-cover pointer-events-none"
                      />

                      {/* Индикатор порядка слева направо: 1, 2, 3... */}
                      <div className="absolute top-2 left-2 flex items-center gap-1 z-10">
                        {idx === 0 ? (
                          <span className="px-2 py-0.5 rounded-md bg-teal-600 text-white text-[11px] font-extrabold shadow-sm flex items-center gap-1">
                            <Star className="w-3 h-3 fill-current" />
                            <span>1 • Обложка</span>
                          </span>
                        ) : (
                          <span className="px-2 py-0.5 rounded-md bg-slate-900/80 backdrop-blur-xs text-white text-[11px] font-bold shadow-sm">
                            {idx + 1}
                          </span>
                        )}
                      </div>

                      {/* Иконка перетаскивания (Grip) */}
                      <div className="absolute top-2 right-8 p-1 rounded-md bg-slate-900/60 text-white opacity-0 group-hover:opacity-100 transition-opacity z-10">
                        <GripVertical className="w-3.5 h-3.5" />
                      </div>

                      {/* Кнопка удаления фото */}
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleRemovePhoto(idx);
                        }}
                        className="absolute top-2 right-2 p-1 rounded-full bg-slate-900/70 hover:bg-rose-600 text-white transition-colors cursor-pointer z-10"
                        title="Удалить фото"
                      >
                        <X className="w-3.5 h-3.5" />
                      </button>

                      {/* Панель быстрого управления при hover внизу карточки */}
                      <div className="absolute inset-x-0 bottom-0 p-2 bg-gradient-to-t from-slate-900/90 via-slate-900/60 to-transparent flex items-center justify-between opacity-0 group-hover:opacity-100 transition-opacity z-10 text-white">
                        <div className="flex items-center gap-1">
                          {idx > 0 && (
                            <button
                              type="button"
                              onClick={(e) => {
                                e.stopPropagation();
                                handleMovePhoto(idx, idx - 1);
                              }}
                              className="p-1 rounded bg-white/20 hover:bg-white/40 text-white transition-colors cursor-pointer"
                              title="Сдвинуть влево"
                            >
                              <ChevronLeft className="w-3.5 h-3.5" />
                            </button>
                          )}
                          {idx < photos.length - 1 && (
                            <button
                              type="button"
                              onClick={(e) => {
                                e.stopPropagation();
                                handleMovePhoto(idx, idx + 1);
                              }}
                              className="p-1 rounded bg-white/20 hover:bg-white/40 text-white transition-colors cursor-pointer"
                              title="Сдвинуть вправо"
                            >
                              <ChevronRight className="w-3.5 h-3.5" />
                            </button>
                          )}
                        </div>

                        {idx > 0 && (
                          <button
                            type="button"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleMakeCoverPhoto(idx);
                            }}
                            className="px-2 py-0.5 rounded bg-teal-600/90 hover:bg-teal-600 text-[10px] font-bold text-white transition-colors cursor-pointer"
                            title="Сделать обложкой"
                          >
                            Сделать обложкой
                          </button>
                        )}
                      </div>
                    </div>
                  );
                })}

                {/* Дополнительная ячейка "Добавить еще" прямо в сетке */}
                {photos.length < 10 && (
                  <div
                    onClick={() => fileInputRef.current?.click()}
                    className="border-2 border-dashed border-slate-200 hover:border-teal-400 hover:bg-teal-50/30 rounded-xl flex flex-col items-center justify-center p-4 text-center cursor-pointer transition-colors aspect-4/3 text-slate-500 hover:text-teal-700 group"
                  >
                    <Plus className="w-6 h-6 mb-1.5 group-hover:scale-110 transition-transform text-teal-600" />
                    <span className="text-xs font-bold">Добавить фото</span>
                    <span className="text-[10px] text-slate-400 mt-0.5">{10 - photos.length} ост.</span>
                  </div>
                )}
              </div>
            ) : (
              <div
                onClick={() => fileInputRef.current?.click()}
                className="p-6 border-2 border-dashed border-slate-200 hover:border-teal-400 bg-white rounded-xl text-center cursor-pointer transition-colors"
              >
                <Upload className="w-6 h-6 text-slate-400 mx-auto mb-1.5" />
                <span className="text-xs font-bold text-slate-700">
                  Нажмите, чтобы загрузить фотографии лота
                </span>
                <p className="text-[11px] text-slate-400 mt-0.5">
                  Первое фото станет главной обложкой карточки. Вы сможете менять порядок карточек перетаскиванием.
                </p>
              </div>
            )}
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
                    : 'Лот сразу поступит на модерацию (Pending Review) для автоматической проверки.'}
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
              disabled={submitting || uploadingPhoto}
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
                      : 'Отправить лот на проверку'}
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
              • Фотографий прикреплено:{' '}
              <strong className="text-teal-700">{photos.length}</strong>
            </div>
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
