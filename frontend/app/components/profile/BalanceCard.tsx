import { useState, useRef } from 'react';
import {
  Coins,
  ShieldAlert,
  Sparkles,
  LogOut,
  CheckCircle2,
  Edit2,
  Check,
  X,
  Camera,
  Laptop,
  Trash2,
  Loader2,
  AlertTriangle,
  Copy,
} from 'lucide-react';
import type { User, UserSession } from '~/api/types';
import { formatCoins } from '~/lib/money';
import { updateUserName, uploadAvatar, getSessions, deleteCurrentUser } from '~/api/auth';
import { resolveAvatarUrl } from '~/api/real/auth';
import { useAuth } from '~/hooks/useAuth';
import { ErrorAlert } from '~/components/feedback/ErrorAlert';

interface BalanceCardProps {
  user: User;
  onClaimDaily: () => Promise<unknown>;
  onLogout: () => void;
}

export function BalanceCard({ user, onClaimDaily, onLogout }: BalanceCardProps) {
  const { refreshUser } = useAuth();

  const [claiming, setClaiming] = useState(false);
  const [claimed, setClaimed] = useState(false);
  const [claimMessage, setClaimMessage] = useState<{ text: string; isError: boolean } | null>(null);

  // Смена имени
  const [isEditingName, setIsEditingName] = useState(false);
  const [editNameValue, setEditNameValue] = useState(user.name);
  const [savingName, setSavingName] = useState(false);
  const [nameError, setNameError] = useState<string | null>(null);

  // Копирование UUID
  const [copiedId, setCopiedId] = useState(false);

  const handleCopyId = async () => {
    if (!user?.id) return;
    try {
      await navigator.clipboard.writeText(user.id);
      setCopiedId(true);
      setTimeout(() => setCopiedId(false), 2000);
    } catch {
      // fallback
    }
  };

  // Загрузка аватара
  const avatarInputRef = useRef<HTMLInputElement>(null);
  const [uploadingAvatar, setUploadingAvatar] = useState(false);
  const [avatarError, setAvatarError] = useState<string | null>(null);
  const [avatarImgFailed, setAvatarImgFailed] = useState(false);
  const [showLogoutModal, setShowLogoutModal] = useState(false);

  // Сессии
  const [showSessionsModal, setShowSessionsModal] = useState(false);
  const [sessions, setSessions] = useState<UserSession[]>([]);
  const [loadingSessions, setLoadingSessions] = useState(false);

  // Удаление аккаунта
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const handleClaim = async () => {
    setClaiming(true);
    setClaimMessage(null);
    try {
      await onClaimDaily();
      setClaimed(true);
      setClaimMessage({ text: '+1 000 🪙 успешно начислено на ваш баланс!', isError: false });
      setTimeout(() => {
        setClaimed(false);
        setClaimMessage(null);
      }, 5000);
    } catch (err: any) {
      const msg = err?.message || 'Не удалось получить бесплатные монеты';
      setClaimMessage({ text: msg, isError: true });
      setTimeout(() => setClaimMessage(null), 7000);
    } finally {
      setClaiming(false);
    }
  };

  const handleSaveName = async () => {
    if (!editNameValue.trim()) {
      setNameError('Имя не может быть пустым');
      return;
    }
    setSavingName(true);
    setNameError(null);
    try {
      await updateUserName(editNameValue.trim());
      await refreshUser();
      setIsEditingName(false);
    } catch (err: unknown) {
      setNameError(err instanceof Error ? err.message : 'Не удалось обновить имя');
    } finally {
      setSavingName(false);
    }
  };

  const handleAvatarFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (file.size > 5 * 1024 * 1024) {
      setAvatarError('Размер фотографии пользователя не должен превышать 5 МБ');
      return;
    }
    setUploadingAvatar(true);
    setAvatarError(null);
    setAvatarImgFailed(false);
    try {
      await uploadAvatar(file);
      await refreshUser();
    } catch (err: unknown) {
      setAvatarError(err instanceof Error ? err.message : 'Не удалось загрузить аватар');
    } finally {
      setUploadingAvatar(false);
    }
  };

  const handleOpenSessions = async () => {
    setShowSessionsModal(true);
    setLoadingSessions(true);
    try {
      const data = await getSessions(false);
      setSessions(Array.isArray(data) ? data : []);
    } catch {
      setSessions([]);
    } finally {
      setLoadingSessions(false);
    }
  };

  const handleDeleteAccount = async () => {
    setDeleting(true);
    try {
      await deleteCurrentUser();
      onLogout();
    } catch (err: unknown) {
      setAvatarError(err instanceof Error ? err.message : 'Не удалось удалить аккаунт');
      setShowDeleteModal(false);
    } finally {
      setDeleting(false);
    }
  };

  const available = user.balance - user.hold;
  const resolvedAvatar = resolveAvatarUrl(user.avatar_id);

  return (
    <div className="bg-white rounded-2xl border border-slate-200/80 p-5 shadow-xs space-y-4">
      {avatarError ? <ErrorAlert error={avatarError} onDismiss={() => setAvatarError(null)} /> : null}

      {/* Пользователь и аватар */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex items-center gap-3.5">
          {/* Аватарка с кнопкой загрузки */}
          <div className="relative group shrink-0">
            {resolvedAvatar && !avatarImgFailed ? (
              <img
                src={resolvedAvatar}
                alt={user.name}
                onError={() => setAvatarImgFailed(true)}
                className="w-13 h-13 rounded-2xl object-cover shadow-xs border border-slate-200"
              />
            ) : (
              <div className="w-13 h-13 rounded-2xl bg-teal-600 text-white font-black text-xl flex items-center justify-center shadow-xs">
                {user.name.charAt(0).toUpperCase()}
              </div>
            )}
            <button
              type="button"
              onClick={() => avatarInputRef.current?.click()}
              disabled={uploadingAvatar}
              className="absolute inset-0 bg-slate-900/40 rounded-2xl opacity-0 group-hover:opacity-100 flex items-center justify-center text-white transition-opacity cursor-pointer disabled:opacity-100"
              title="Изменить аватар"
            >
              {uploadingAvatar ? (
                <Loader2 className="w-5 h-5 animate-spin" />
              ) : (
                <Camera className="w-5 h-5" />
              )}
            </button>
            <input
              ref={avatarInputRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={handleAvatarFileChange}
            />
          </div>

          <div>
            {isEditingName ? (
              <div className="flex items-center gap-1.5 mt-0.5">
                <input
                  type="text"
                  value={editNameValue}
                  onChange={(e) => setEditNameValue(e.target.value)}
                  className="px-2.5 py-1 text-sm font-semibold rounded-lg border border-teal-500 focus:outline-none focus:ring-1 focus:ring-teal-500 text-slate-900"
                  autoFocus
                />
                <button
                  type="button"
                  onClick={handleSaveName}
                  disabled={savingName}
                  className="p-1 rounded-lg bg-teal-600 hover:bg-teal-700 text-white transition-colors cursor-pointer"
                  title="Сохранить"
                >
                  {savingName ? <Loader2 className="w-4 h-4 animate-spin" /> : <Check className="w-4 h-4" />}
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setIsEditingName(false);
                    setEditNameValue(user.name);
                  }}
                  className="p-1 rounded-lg bg-slate-100 hover:bg-slate-200 text-slate-600 transition-colors cursor-pointer"
                  title="Отмена"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            ) : (
              <div className="flex items-center gap-2">
                <h2 className="text-base sm:text-lg font-bold text-slate-900">{user.name}</h2>
                <button
                  type="button"
                  onClick={() => {
                    setEditNameValue(user.name);
                    setIsEditingName(true);
                  }}
                  className="p-1 text-slate-400 hover:text-slate-600 rounded-lg hover:bg-slate-100 transition-colors cursor-pointer"
                  title="Редактировать имя"
                >
                  <Edit2 className="w-3.5 h-3.5" />
                </button>
              </div>
            )}
            {nameError ? <p className="text-[11px] text-rose-600 mt-0.5">{nameError}</p> : null}
            <p className="text-xs text-slate-400 mt-0.5">{user.email}</p>

            {/* UUID пользователя для получения подарков */}
            {user.id && (
              <div className="flex flex-wrap items-center gap-2 mt-2">
                <div className="inline-flex items-center gap-1.5 px-2.5 py-1 bg-slate-50 hover:bg-slate-100/80 border border-slate-200/80 rounded-lg text-xs font-mono transition-colors">
                  <span className="text-[11px] font-sans text-slate-500 font-medium">Ваш UUID:</span>
                  <span className="font-semibold text-slate-800 select-all">{user.id}</span>
                  <button
                    type="button"
                    onClick={handleCopyId}
                    className="p-1 hover:text-teal-700 text-slate-400 hover:bg-white rounded transition-colors cursor-pointer"
                    title="Скопировать ваш UUID для получения подарков"
                  >
                    {copiedId ? (
                      <Check className="w-3.5 h-3.5 text-emerald-600" />
                    ) : (
                      <Copy className="w-3.5 h-3.5" />
                    )}
                  </button>
                </div>
                {copiedId && (
                  <span className="text-[11px] font-semibold text-emerald-600">
                    Скопировано!
                  </span>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Кнопки управления аккаунтом */}
        <div className="flex items-center gap-2 self-start sm:self-auto">
          <button
            type="button"
            onClick={handleOpenSessions}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs text-slate-600 hover:text-teal-700 bg-slate-50 hover:bg-teal-50 border border-slate-200 rounded-xl transition-colors font-medium cursor-pointer"
          >
            <Laptop className="w-3.5 h-3.5" />
            <span>Сессии</span>
          </button>

          <button
            type="button"
            onClick={() => setShowLogoutModal(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs text-slate-500 hover:text-rose-600 hover:bg-rose-50 border border-slate-200 rounded-xl transition-colors font-medium cursor-pointer"
          >
            <LogOut className="w-3.5 h-3.5" />
            <span>Выйти</span>
          </button>

          <button
            type="button"
            onClick={() => setShowDeleteModal(true)}
            className="p-1.5 text-slate-400 hover:text-rose-600 hover:bg-rose-50 rounded-xl transition-colors cursor-pointer"
            title="Удалить аккаунт"
          >
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Блок балансов */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 pt-3 border-t border-slate-100">
        {/* Баланс Coins */}
        <div className="group relative p-3 rounded-xl bg-amber-50/70 border border-amber-200/70 cursor-help">
          <div className="flex items-center gap-1 text-[11px] font-semibold text-amber-800">
            <Coins className="w-3.5 h-3.5 text-amber-600" />
            <span>Баланс монет</span>
          </div>
          <div className="text-lg font-black text-amber-950 mt-1">
            {formatCoins(user.balance)}
          </div>
          <div className="absolute left-0 bottom-full mb-1 hidden group-hover:block p-2 bg-slate-900 text-white text-[11px] rounded-lg shadow-lg z-50 pointer-events-none w-52">
            Пополняется автоматически при ежедневном заходе на сайт.
          </div>
        </div>

        {/* Доступно */}
        <div className="p-3 rounded-xl bg-teal-50/70 border border-teal-200/70">
          <div className="text-[11px] font-semibold text-teal-800">Доступно для ставок</div>
          <div className="text-lg font-black text-teal-950 mt-1">
            {formatCoins(available)}
          </div>
        </div>

        {/* Заморожено (Hold) */}
        <div className="group relative p-3 rounded-xl bg-slate-100/80 border border-slate-200/80 cursor-help">
          <div className="flex items-center gap-1 text-[11px] font-semibold text-slate-700">
            <ShieldAlert className="w-3.5 h-3.5 text-slate-500" />
            <span>Заморожено (Hold)</span>
          </div>
          <div className="text-lg font-black text-slate-800 mt-1">
            {formatCoins(user.hold)}
          </div>
          <div className="absolute left-0 bottom-full mb-1 hidden group-hover:block p-2 bg-slate-900 text-white text-[11px] rounded-lg shadow-lg z-50 pointer-events-none w-60">
            Замороженные средства для активных ставок. Размораживаются после перебивания или завершения.
          </div>
        </div>
      </div>

      {/* Ежедневный бонус */}
      <div className="pt-2 space-y-2">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1.5 text-xs text-slate-500">
            <Sparkles className="w-4 h-4 text-amber-500" />
            <span>Бесплатные монеты (freeCoin): <strong>+1 000 🪙</strong></span>
          </div>

          <button
            type="button"
            onClick={handleClaim}
            disabled={claiming}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-amber-500 hover:bg-amber-600 active:scale-98 text-white text-xs font-bold rounded-xl shadow-xs transition-all disabled:opacity-50 cursor-pointer"
          >
            {claimed ? (
              <>
                <CheckCircle2 className="w-3.5 h-3.5" />
                <span>Получено!</span>
              </>
            ) : (
              <>
                <Coins className="w-3.5 h-3.5" />
                <span>{claiming ? 'Начисление...' : 'Получить +1 000 🪙'}</span>
              </>
            )}
          </button>
        </div>

        {claimMessage && (
          <div
            className={`p-2.5 rounded-xl text-xs flex items-center gap-2 transition-all ${
              claimMessage.isError
                ? 'bg-amber-50 border border-amber-200 text-amber-900'
                : 'bg-emerald-50 border border-emerald-200 text-emerald-900'
            }`}
          >
            {claimMessage.isError ? (
              <AlertTriangle className="w-4 h-4 shrink-0 text-amber-600" />
            ) : (
              <CheckCircle2 className="w-4 h-4 shrink-0 text-emerald-600" />
            )}
            <span className="font-medium">{claimMessage.text}</span>
          </div>
        )}
      </div>

      {/* Модальное окно подтверждения выхода */}
      {showLogoutModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/40 backdrop-blur-xs">
          <div className="bg-white rounded-2xl border border-slate-200 p-5 sm:p-6 w-full max-w-sm shadow-2xl space-y-4 animate-in fade-in zoom-in-95">
            <div className="flex items-center gap-3 text-slate-800">
              <div className="w-10 h-10 rounded-xl bg-slate-100 flex items-center justify-center shrink-0 text-slate-600">
                <LogOut className="w-5 h-5" />
              </div>
              <div>
                <h3 className="font-bold text-sm sm:text-base text-slate-900">Выход из аккаунта</h3>
                <p className="text-xs text-slate-500 mt-0.5">Предотвращение случайного нажатия</p>
              </div>
            </div>

            <p className="text-xs text-slate-600 leading-relaxed">
              Вы уверены, что хотите выйти из профиля <strong>{user.name}</strong>?
            </p>

            <div className="pt-3 border-t border-slate-100 flex items-center justify-end gap-2">
              <button
                type="button"
                onClick={() => setShowLogoutModal(false)}
                className="px-3.5 py-2 bg-slate-100 hover:bg-slate-200 text-slate-700 text-xs font-semibold rounded-xl transition-colors cursor-pointer"
              >
                Отмена
              </button>
              <button
                type="button"
                onClick={() => {
                  setShowLogoutModal(false);
                  onLogout();
                }}
                className="px-4 py-2 bg-rose-600 hover:bg-rose-700 text-white text-xs font-bold rounded-xl shadow-xs transition-colors flex items-center gap-1.5 cursor-pointer"
              >
                <LogOut className="w-3.5 h-3.5" />
                <span>Да, выйти</span>
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Модальное окно активных сессий */}
      {showSessionsModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/40 backdrop-blur-xs">
          <div className="bg-white rounded-2xl border border-slate-200 p-5 sm:p-6 w-full max-w-md shadow-2xl space-y-4 animate-in fade-in zoom-in-95">
            <div className="flex items-center justify-between pb-3 border-b border-slate-100">
              <div className="flex items-center gap-2">
                <Laptop className="w-4.5 h-4.5 text-teal-600" />
                <h3 className="font-bold text-sm sm:text-base text-slate-900">Активные сессии</h3>
              </div>
              <button
                type="button"
                onClick={() => setShowSessionsModal(false)}
                className="p-1 rounded-lg text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors cursor-pointer"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {loadingSessions ? (
              <div className="py-8 flex flex-col items-center justify-center space-y-2">
                <Loader2 className="w-6 h-6 text-teal-600 animate-spin" />
                <span className="text-xs text-slate-400">Загрузка сессий...</span>
              </div>
            ) : sessions.length === 0 ? (
              <p className="text-xs text-slate-400 text-center py-6">Нет данных о сессиях</p>
            ) : (
              <div className="divide-y divide-slate-100 max-h-64 overflow-y-auto space-y-2 pr-1">
                {sessions.map((sess, idx) => (
                  <div key={sess.refresh_id || idx} className="pt-2 text-xs flex justify-between items-center">
                    <div>
                      <p className="font-semibold text-slate-800">
                        {sess.device?.browser || 'Браузер'} • {sess.device?.os || 'Система'}
                      </p>
                      <p className="text-[10px] text-slate-400 mt-0.5">
                        Создана: {new Date(sess.created_at).toLocaleDateString()}
                      </p>
                    </div>
                    {sess.revoked_at ? (
                      <span className="text-[10px] text-slate-400 bg-slate-100 px-2 py-0.5 rounded-full">
                        Завершена
                      </span>
                    ) : (
                      <span className="text-[10px] text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded-full font-semibold">
                        Активна
                      </span>
                    )}
                  </div>
                ))}
              </div>
            )}

            <div className="pt-3 border-t border-slate-100 flex justify-end">
              <button
                type="button"
                onClick={() => setShowSessionsModal(false)}
                className="px-4 py-2 bg-slate-100 hover:bg-slate-200 text-slate-700 text-xs font-semibold rounded-xl transition-colors cursor-pointer"
              >
                Закрыть
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Модальное окно подтверждения удаления аккаунта */}
      {showDeleteModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/40 backdrop-blur-xs">
          <div className="bg-white rounded-2xl border border-rose-200 p-5 sm:p-6 w-full max-w-sm shadow-2xl space-y-4 animate-in fade-in zoom-in-95">
            <div className="flex items-center gap-3 text-rose-600">
              <div className="w-10 h-10 rounded-xl bg-rose-50 flex items-center justify-center shrink-0">
                <AlertTriangle className="w-5 h-5" />
              </div>
              <div>
                <h3 className="font-bold text-sm sm:text-base text-slate-900">Удаление аккаунта</h3>
                <p className="text-xs text-slate-500 mt-0.5">Это действие необратимо</p>
              </div>
            </div>

            <p className="text-xs text-slate-600 leading-relaxed">
              Вы уверены, что хотите удалить свою учетную запись? Все ваши персональные данные будут
              безвозвратно удалены с сервера.
            </p>

            <div className="pt-3 border-t border-slate-100 flex items-center justify-end gap-2">
              <button
                type="button"
                onClick={() => setShowDeleteModal(false)}
                disabled={deleting}
                className="px-3 py-1.5 bg-slate-100 hover:bg-slate-200 text-slate-700 text-xs font-semibold rounded-xl transition-colors cursor-pointer"
              >
                Отмена
              </button>
              <button
                type="button"
                onClick={handleDeleteAccount}
                disabled={deleting}
                className="px-4 py-1.5 bg-rose-600 hover:bg-rose-700 text-white text-xs font-bold rounded-xl shadow-xs transition-colors flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
              >
                {deleting ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : null}
                <span>Удалить навсегда</span>
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
