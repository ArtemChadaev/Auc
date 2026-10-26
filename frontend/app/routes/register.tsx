import { useState, useRef, useEffect } from 'react';
import { Link, useNavigate } from 'react-router';
import {
  Check,
  ArrowRight,
  ArrowLeft,
  Loader2,
  FileText,
  X,
  Upload,
  Image as ImageIcon,
  CheckCircle2,
  CloudUpload,
} from 'lucide-react';
import { useAuth } from '~/hooks/useAuth';
import { listDocuments, recordConsent } from '~/api/docs';
import { uploadAvatar } from '~/api/auth';
import type { Document } from '~/api/types';
import { ErrorAlert } from '~/components/feedback/ErrorAlert';

export const PRESET_AVATARS = [
  { id: 'avatar-1', name: 'Лиса-кодер', color: 'bg-orange-500' },
  { id: 'avatar-2', name: 'Кот-моделлер', color: 'bg-emerald-500' },
  { id: 'avatar-3', name: 'Медведь-архитектор', color: 'bg-blue-500' },
  { id: 'avatar-4', name: 'Сова-дизайнер', color: 'bg-purple-500' },
  { id: 'avatar-5', name: 'Волк-хакер', color: 'bg-slate-700' },
];

const DEFAULT_DOCUMENTS: Document[] = [
  {
    id: 1,
    name: 'terms-of-service',
    version: '1.0',
    is_major: true,
    published_at: new Date().toISOString(),
    text: `# Пользовательское соглашение\n\n1. Предмет соглашения\nНастоящая платформа предназначена для аукциона цифровых файлов и виртуальных предметов.\n\n2. Расчеты\nВсе расчеты производятся в виртуальных Coins.\n\n3. Правила ставок и Hold\nПри каждой ставке сумма замораживается на счёте покупателя (Hold).`,
  },
  {
    id: 2,
    name: 'privacy-policy',
    version: '1.0',
    is_major: true,
    published_at: new Date().toISOString(),
    text: `# Политика конфиденциальности\n\n1. Обработка данных\nПлатформа обрабатывает минимально необходимые данные для аутентификации.\n\n2. Безопасность\nПароли хранятся в виде криптографических хешей.`,
  },
];

export default function RegisterPage() {
  const [step, setStep] = useState<1 | 2 | 3>(1);

  // Шаг 1: Учётные данные
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [passwordConfirm, setPasswordConfirm] = useState('');

  // Шаг 2: Согласия
  const [termsAgreed, setTermsAgreed] = useState(false);
  const [privacyAgreed, setPrivacyAgreed] = useState(false);
  const [activeDocModal, setActiveDocModal] = useState<string | null>(null);
  const [documents, setDocuments] = useState<Document[]>(DEFAULT_DOCUMENTS);

  // Шаг 3: Аватар (пресет или кастомный)
  const [selectedAvatar, setSelectedAvatar] = useState('avatar-1');
  const [customAvatarUrl, setCustomAvatarUrl] = useState<string | null>(null);
  const [customAvatarFile, setCustomAvatarFile] = useState<File | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Модалка степени загрузки данных на сервер
  const [showUploadModal, setShowUploadModal] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [uploadStatusText, setUploadStatusText] = useState('Подготовка к отправке...');
  const [uploadDone, setUploadDone] = useState(false);

  const [error, setError] = useState<string | null>(null);

  const { register } = useAuth();
  const navigate = useNavigate();

  // Валидация шага 1
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    listDocuments()
      .then((docs) => {
        if (Array.isArray(docs) && docs.length > 0) {
          setDocuments(docs);
        }
      })
      .catch(() => {});
  }, []);

  // Валидация шага 1
  const handleNextStep1 = (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    const errors: Record<string, string> = {};

    if (!name.trim()) {
      errors.name = 'Пожалуйста, введите ваше имя';
    } else if (name.trim().length < 2) {
      errors.name = 'Имя должно содержать не менее 2 символов';
    }

    if (!email.trim() || !email.includes('@')) {
      errors.email = 'Пожалуйста, введите корректный email адрес';
    }

    if (!password) {
      errors.password = 'Пожалуйста, укажите пароль';
    } else if (password.length < 6) {
      errors.password = 'Пароль должен содержать не менее 6 символов';
    }

    if (!passwordConfirm) {
      errors.passwordConfirm = 'Пожалуйста, подтвердите пароль';
    } else if (password !== passwordConfirm) {
      errors.passwordConfirm = 'Пароли не совпадают';
    }

    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      return;
    }

    setFieldErrors({});
    setStep(2);
  };

  // Валидация шага 2
  const handleNextStep2 = (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    const errors: Record<string, string> = {};

    if (!termsAgreed || !privacyAgreed) {
      errors.consent = 'Для продолжения необходимо принять оба соглашения';
      setFieldErrors(errors);
      return;
    }

    setFieldErrors({});
    setStep(3);
  };

  // Обработка загрузки своей фотографии
  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    if (!file.type.startsWith('image/')) {
      setError('Пожалуйста, выберите файл изображения (PNG, JPG, WEBP)');
      return;
    }

    setCustomAvatarFile(file);
    const reader = new FileReader();
    reader.onload = (event) => {
      if (typeof event.target?.result === 'string') {
        setCustomAvatarUrl(event.target.result);
        setSelectedAvatar('custom');
      }
    };
    reader.readAsDataURL(file);
  };

  // Финальная отправка с пошаговой синхронизацией с сервером
  const handleFinalSubmit = async () => {
    setShowUploadModal(true);
    setUploadProgress(15);
    setUploadStatusText('Создание учетной записи на сервере...');

    try {
      // 1. Регистрация пользователя POST /api/auth/register
      await register(name.trim(), email.trim(), selectedAvatar, password);

      setUploadProgress(45);
      setUploadStatusText('Фиксация согласий с документами...');

      // 2. Фиксация согласий POST /api/document/consents
      const termsDoc = documents.find((d) => d.name.includes('terms') || d.id === 1);
      const privacyDoc = documents.find((d) => d.name.includes('privacy') || d.id === 2);

      if (termsDoc?.id) {
        await recordConsent(termsDoc.id).catch(() => {});
      }
      if (privacyDoc?.id) {
        await recordConsent(privacyDoc.id).catch(() => {});
      }

      setUploadProgress(70);

      // 3. Загрузка аватара при наличии (PUT /api/user/me/avatar)
      if (customAvatarFile) {
        setUploadStatusText('Загрузка аватара в S3 хранилище...');
        try {
          await uploadAvatar(customAvatarFile);
        } catch {
          // Игнорируем ошибку аватара, регистрация уже прошла
        }
      }

      setUploadProgress(100);
      setUploadStatusText('Регистрация успешно завершена!');
      setUploadDone(true);

      setTimeout(() => {
        setShowUploadModal(false);
        navigate('/lots');
      }, 1000);
    } catch (err: unknown) {
      setShowUploadModal(false);
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError('Ошибка при отправке данных на сервер');
      }
    }
  };

  const currentDoc = documents.find(
    (d) => d.name === activeDocModal || d.name.includes(activeDocModal || '')
  );

  return (
    <div className="max-w-md mx-auto py-6 sm:py-10 space-y-5">
      <div className="text-center space-y-1">
        <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-slate-900">Регистрация</h1>
        <p className="text-xs text-slate-500">Создание аккаунта за 3 простых шага</p>
      </div>

      {/* Индикатор этапов */}
      <div className="flex items-center justify-between px-4">
        {[
          { num: 1, label: 'Данные' },
          { num: 2, label: 'Согласия' },
          { num: 3, label: 'Аватарка' },
        ].map((s, idx) => (
          <div key={s.num} className="flex items-center gap-2">
            <div
              className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold transition-colors ${
                step === s.num
                  ? 'bg-teal-600 text-white shadow-xs'
                  : step > s.num
                  ? 'bg-emerald-500 text-white'
                  : 'bg-slate-200 text-slate-500'
              }`}
            >
              {step > s.num ? <Check className="w-3.5 h-3.5" /> : s.num}
            </div>
            <span
              className={`text-xs font-medium hidden sm:inline ${
                step === s.num ? 'text-slate-900 font-bold' : 'text-slate-400'
              }`}
            >
              {s.label}
            </span>
            {idx < 2 && <div className="w-8 sm:w-12 h-0.5 bg-slate-200 mx-1" />}
          </div>
        ))}
      </div>

      <div className="bg-white rounded-2xl border border-slate-200/80 p-5 sm:p-6 shadow-xs space-y-4">
        {error ? <ErrorAlert error={error} onDismiss={() => setError(null)} /> : null}

        {/* ШАГ 1: Учётные данные */}
        {step === 1 && (
          <form onSubmit={handleNextStep1} noValidate className="space-y-3.5">
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">Имя</label>
              <input
                type="text"
                value={name}
                onChange={(e) => {
                  setName(e.target.value);
                  if (fieldErrors.name) setFieldErrors((prev) => ({ ...prev, name: '' }));
                }}
                placeholder="Иван Петров"
                className={`w-full px-3.5 py-2.5 text-xs sm:text-sm bg-slate-50 border rounded-xl focus:outline-hidden focus:ring-2 ${
                  fieldErrors.name
                    ? 'border-rose-400 bg-rose-50/20 focus:ring-rose-400/20'
                    : 'border-slate-200 focus:ring-teal-500/20 focus:border-teal-500'
                }`}
              />
              {fieldErrors.name ? (
                <p className="mt-1 text-xs text-rose-600 font-medium">{fieldErrors.name}</p>
              ) : null}
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">
                Электронная почта
              </label>
              <input
                type="email"
                value={email}
                onChange={(e) => {
                  setEmail(e.target.value);
                  if (fieldErrors.email) setFieldErrors((prev) => ({ ...prev, email: '' }));
                }}
                placeholder="ivan@example.com"
                className={`w-full px-3.5 py-2.5 text-xs sm:text-sm bg-slate-50 border rounded-xl focus:outline-hidden focus:ring-2 ${
                  fieldErrors.email
                    ? 'border-rose-400 bg-rose-50/20 focus:ring-rose-400/20'
                    : 'border-slate-200 focus:ring-teal-500/20 focus:border-teal-500'
                }`}
              />
              {fieldErrors.email ? (
                <p className="mt-1 text-xs text-rose-600 font-medium">{fieldErrors.email}</p>
              ) : null}
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">Пароль</label>
              <input
                type="password"
                value={password}
                onChange={(e) => {
                  setPassword(e.target.value);
                  if (fieldErrors.password) setFieldErrors((prev) => ({ ...prev, password: '' }));
                }}
                placeholder="Минимум 6 символов"
                className={`w-full px-3.5 py-2.5 text-xs sm:text-sm bg-slate-50 border rounded-xl focus:outline-hidden focus:ring-2 ${
                  fieldErrors.password
                    ? 'border-rose-400 bg-rose-50/20 focus:ring-rose-400/20'
                    : 'border-slate-200 focus:ring-teal-500/20 focus:border-teal-500'
                }`}
              />
              {fieldErrors.password ? (
                <p className="mt-1 text-xs text-rose-600 font-medium">{fieldErrors.password}</p>
              ) : null}
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">
                Подтверждение пароля
              </label>
              <input
                type="password"
                value={passwordConfirm}
                onChange={(e) => {
                  setPasswordConfirm(e.target.value);
                  if (fieldErrors.passwordConfirm)
                    setFieldErrors((prev) => ({ ...prev, passwordConfirm: '' }));
                }}
                placeholder="Повторите пароль"
                className={`w-full px-3.5 py-2.5 text-xs sm:text-sm bg-slate-50 border rounded-xl focus:outline-hidden focus:ring-2 ${
                  fieldErrors.passwordConfirm
                    ? 'border-rose-400 bg-rose-50/20 focus:ring-rose-400/20'
                    : 'border-slate-200 focus:ring-teal-500/20 focus:border-teal-500'
                }`}
              />
              {fieldErrors.passwordConfirm ? (
                <p className="mt-1 text-xs text-rose-600 font-medium">
                  {fieldErrors.passwordConfirm}
                </p>
              ) : null}
            </div>

            <button
              type="submit"
              className="w-full py-2.5 bg-teal-600 hover:bg-teal-700 text-white font-bold text-xs sm:text-sm rounded-xl shadow-xs transition-colors flex items-center justify-center gap-1.5 cursor-pointer mt-2"
            >
              <span>Далее: Согласия</span>
              <ArrowRight className="w-4 h-4" />
            </button>
          </form>
        )}

        {/* ШАГ 2: Согласия с документами */}
        {step === 2 && (
          <form onSubmit={handleNextStep2} noValidate className="space-y-4">
            <p className="text-xs text-slate-500">
              Пожалуйста, ознакомьтесь с правилами платформы и подтвердите согласие:
            </p>

            {fieldErrors.consent ? (
              <div className="p-3 bg-rose-50 border border-rose-200 text-rose-800 text-xs font-medium rounded-xl">
                {fieldErrors.consent}
              </div>
            ) : null}

            <div className="space-y-3 p-3.5 bg-slate-50 rounded-xl border border-slate-200/80">
              <label className="flex items-start gap-3 cursor-pointer text-xs">
                <input
                  type="checkbox"
                  checked={termsAgreed}
                  onChange={(e) => setTermsAgreed(e.target.checked)}
                  className="mt-0.5 w-4 h-4 text-teal-600 rounded border-slate-300 focus:ring-teal-500"
                />
                <span className="text-slate-700 leading-relaxed">
                  Я согласен с{' '}
                  <button
                    type="button"
                    onClick={() => setActiveDocModal('terms')}
                    className="text-teal-600 underline font-semibold hover:text-teal-700 cursor-pointer"
                  >
                    Пользовательским соглашением
                  </button>
                </span>
              </label>

              <label className="flex items-start gap-3 cursor-pointer text-xs">
                <input
                  type="checkbox"
                  checked={privacyAgreed}
                  onChange={(e) => setPrivacyAgreed(e.target.checked)}
                  className="mt-0.5 w-4 h-4 text-teal-600 rounded border-slate-300 focus:ring-teal-500"
                />
                <span className="text-slate-700 leading-relaxed">
                  Я согласен с{' '}
                  <button
                    type="button"
                    onClick={() => setActiveDocModal('privacy')}
                    className="text-teal-600 underline font-semibold hover:text-teal-700 cursor-pointer"
                  >
                    Политикой конфиденциальности
                  </button>
                </span>
              </label>
            </div>

            <div className="flex items-center gap-3 pt-2">
              <button
                type="button"
                onClick={() => setStep(1)}
                className="py-2.5 px-4 bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold text-xs rounded-xl transition-colors flex items-center gap-1 cursor-pointer"
              >
                <ArrowLeft className="w-3.5 h-3.5" />
                <span>Назад</span>
              </button>
              <button
                type="submit"
                className="flex-1 py-2.5 bg-teal-600 hover:bg-teal-700 text-white font-bold text-xs sm:text-sm rounded-xl shadow-xs transition-colors flex items-center justify-center gap-1.5 cursor-pointer"
              >
                <span>Далее: Выбор аватара</span>
                <ArrowRight className="w-4 h-4" />
              </button>
            </div>
          </form>
        )}

        {/* ШАГ 3: Выбор аватарки (или загрузка своего квадратного фото) */}
        {step === 3 && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <p className="text-xs text-slate-500">
                Выберите аватарку или загрузите своё фото:
              </p>
            </div>

            {/* Скрытый input для выбора файла */}
            <input
              type="file"
              ref={fileInputRef}
              accept="image/*"
              onChange={handleFileChange}
              className="hidden"
            />

            {/* Карточка добавления своего фото (квадратное превью) */}
            <div className="p-3 bg-slate-50 border border-slate-200 rounded-xl space-y-2">
              <span className="text-[11px] font-bold text-slate-600 uppercase tracking-wider block">
                Своя фотография
              </span>

              {customAvatarUrl ? (
                <div className="flex items-center gap-3.5">
                  <div className="relative w-16 h-16 rounded-xl overflow-hidden aspect-square border-2 border-teal-500 shadow-xs shrink-0 bg-slate-200">
                    <img
                      src={customAvatarUrl}
                      alt="Аватар"
                      className="w-full h-full object-cover"
                    />
                  </div>

                  <div className="space-y-1">
                    <div className="flex items-center gap-1 text-xs font-bold text-teal-800">
                      <CheckCircle2 className="w-3.5 h-3.5 text-teal-600" />
                      <span>Квадратное фото готово</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <button
                        type="button"
                        onClick={() => fileInputRef.current?.click()}
                        className="text-xs font-semibold text-teal-600 hover:underline cursor-pointer"
                      >
                        Заменить
                      </button>
                      <button
                        type="button"
                        onClick={() => {
                          setCustomAvatarUrl(null);
                          setCustomAvatarFile(null);
                          setSelectedAvatar('avatar-1');
                        }}
                        className="text-xs text-rose-500 hover:underline cursor-pointer"
                      >
                        Удалить
                      </button>
                    </div>
                  </div>
                </div>
              ) : (
                <button
                  type="button"
                  onClick={() => fileInputRef.current?.click()}
                  className="w-full py-3 px-4 border-2 border-dashed border-slate-300 hover:border-teal-500 hover:bg-teal-50/40 rounded-xl text-center flex items-center justify-center gap-2.5 transition-all cursor-pointer group"
                >
                  <Upload className="w-4 h-4 text-slate-400 group-hover:text-teal-600 transition-colors" />
                  <span className="text-xs font-semibold text-slate-600 group-hover:text-teal-900 transition-colors">
                    Выбрать фото из компьютера (сделаем квадрат)
                  </span>
                </button>
              )}
            </div>

            {/* 5 стандартных аватарок */}
            <div className="space-y-1.5">
              <span className="text-[11px] font-bold text-slate-400 uppercase tracking-wider block">
                Или выберите стандартный аватар:
              </span>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                {PRESET_AVATARS.map((avatar) => {
                  const isSelected = selectedAvatar === avatar.id && !customAvatarUrl;
                  return (
                    <button
                      key={avatar.id}
                      type="button"
                      onClick={() => {
                        setSelectedAvatar(avatar.id);
                        setCustomAvatarUrl(null);
                        setCustomAvatarFile(null);
                      }}
                      className={`flex items-center gap-2.5 p-2.5 rounded-xl border text-left transition-all cursor-pointer ${
                        isSelected
                          ? 'border-teal-500 bg-teal-50/60 ring-2 ring-teal-500/20'
                          : 'border-slate-200/80 hover:bg-slate-50'
                      }`}
                    >
                      <div
                        className={`w-8 h-8 rounded-lg ${avatar.color} text-white font-black text-xs flex items-center justify-center shadow-xs shrink-0`}
                      >
                        {avatar.name.charAt(0)}
                      </div>
                      <div className="min-w-0">
                        <div className="font-semibold text-xs text-slate-900 truncate">
                          {avatar.name}
                        </div>
                      </div>
                    </button>
                  );
                })}
              </div>
            </div>

            <div className="flex items-center gap-3 pt-2">
              <button
                type="button"
                onClick={() => setStep(2)}
                className="py-2.5 px-4 bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold text-xs rounded-xl transition-colors flex items-center gap-1 cursor-pointer"
              >
                <ArrowLeft className="w-3.5 h-3.5" />
                <span>Назад</span>
              </button>
              <button
                type="button"
                onClick={handleFinalSubmit}
                className="flex-1 py-2.5 bg-teal-600 hover:bg-teal-700 text-white font-bold text-xs sm:text-sm rounded-xl shadow-xs transition-colors flex items-center justify-center gap-1.5 cursor-pointer"
              >
                <span>Завершить регистрацию</span>
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Модалка отправки данных на сервер с индикатором прогресса */}
      {showUploadModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs animate-in fade-in">
          <div className="bg-white rounded-2xl border border-slate-200 p-6 w-full max-w-sm shadow-2xl text-center space-y-4">
            <div className="w-14 h-14 rounded-2xl bg-teal-50 text-teal-600 mx-auto flex items-center justify-center">
              {uploadDone ? (
                <CheckCircle2 className="w-8 h-8 text-emerald-600 animate-in zoom-in" />
              ) : (
                <CloudUpload className="w-8 h-8 text-teal-600 animate-pulse" />
              )}
            </div>

            <div>
              <h3 className="font-bold text-base text-slate-900">
                {uploadDone ? 'Данные сохранены!' : 'Отправка данных на сервер'}
              </h3>
              <p className="text-xs text-slate-500 mt-1">{uploadStatusText}</p>
            </div>

            <div className="space-y-1.5">
              <div className="w-full bg-slate-100 rounded-full h-2.5 overflow-hidden">
                <div
                  className="bg-teal-600 h-full rounded-full transition-all duration-300 ease-out"
                  style={{ width: `${uploadProgress}%` }}
                />
              </div>
              <div className="flex justify-between text-[11px] font-semibold text-slate-400">
                <span>Прогресс:</span>
                <span className="text-teal-700">{uploadProgress}%</span>
              </div>
            </div>

            <p className="text-[11px] text-slate-400">
              Пожалуйста, подождите завершения синхронизации с базой данных
            </p>
          </div>
        </div>
      )}

      {/* Модальное окно для чтения документа */}
      {activeDocModal && currentDoc && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/40 backdrop-blur-xs">
          <div className="bg-white rounded-2xl border border-slate-200 p-5 w-full max-w-lg max-h-[80vh] flex flex-col shadow-xl space-y-4 animate-in fade-in">
            <div className="flex items-center justify-between pb-3 border-b border-slate-100">
              <div className="flex items-center gap-2">
                <FileText className="w-4 h-4 text-teal-600" />
                <h3 className="font-bold text-sm text-slate-900">
                  {currentDoc.name.includes('terms')
                    ? 'Пользовательское соглашение'
                    : 'Политика конфиденциальности'}
                </h3>
              </div>
              <button
                onClick={() => setActiveDocModal(null)}
                className="p-1 rounded-lg text-slate-400 hover:text-slate-700 hover:bg-slate-100 cursor-pointer"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="overflow-y-auto flex-1 pr-2 text-xs text-slate-700 leading-relaxed whitespace-pre-line">
              {currentDoc.text}
            </div>

            <div className="pt-2 border-t border-slate-100 text-right">
              <button
                onClick={() => setActiveDocModal(null)}
                className="px-4 py-2 bg-teal-600 text-white font-semibold text-xs rounded-xl shadow-xs cursor-pointer"
              >
                Понятно, закрыть
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
