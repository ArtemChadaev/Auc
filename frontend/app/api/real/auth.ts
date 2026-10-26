import { api, setStoredToken, setStoredRefreshId, getStoredRefreshId, generateIdempotencyKey } from '~/api/client';
import type { User, TokenResponse, UserSession } from '~/api/types';

/**
 * POST /api/auth/login
 * Аутентификация пользователя по email и паролю, возвращает access и refresh токены
 */
export async function loginReal(email: string, password?: string): Promise<TokenResponse> {
  const res = await api.post<TokenResponse>('/api/auth/login', {
    email: email.trim().toLowerCase(),
    password: password || '',
  });

  if (res?.access_token) {
    setStoredToken(res.access_token);
  }
  if (res?.refresh_session?.refresh_id) {
    setStoredRefreshId(res.refresh_session.refresh_id);
  }

  return res;
}

/**
 * POST /api/auth/register
 * Создает новый аккаунт пользователя и возвращает токены
 */
export async function registerReal(
  name: string,
  email: string,
  password?: string
): Promise<TokenResponse> {
  const res = await api.post<TokenResponse>('/api/auth/register', {
    name: name.trim(),
    email: email.trim().toLowerCase(),
    password: password || '',
  });

  if (res?.access_token) {
    setStoredToken(res.access_token);
  }
  if (res?.refresh_session?.refresh_id) {
    setStoredRefreshId(res.refresh_session.refresh_id);
  }

  return res;
}

/**
 * POST /api/auth/logout
 * Завершает указанные refresh-сессии пользователя. Требует авторизации
 */
export async function logoutReal(): Promise<void> {
  const currentRefreshId = getStoredRefreshId();
  try {
    await api.post('/api/auth/logout', {
      refresh_id: currentRefreshId ? [currentRefreshId] : [],
    });
  } catch {
    // Игнорируем сетевые сбои при выходе
  } finally {
    setStoredToken(null);
    setStoredRefreshId(null);
  }
}

/**
 * POST /api/auth/session
 * Возвращает список сессий пользователя (текущих или всех). Требует авторизации
 */
export async function getSessionsReal(current: boolean = false): Promise<UserSession[]> {
  return api.post<UserSession[]>('/api/auth/session', {
    current,
  });
}

/**
 * GET /api/user/me
 * Возвращает данные текущего авторизованного пользователя
 */
export async function getCurrentUserReal(): Promise<User> {
  const user = await api.get<User>('/api/user/me');
  return {
    ...user,
    balance: Number(user.balance ?? 0),
    hold: Number(user.hold ?? 0),
  };
}

/**
 * PATCH /api/user/name
 * Изменяет отображаемое имя текущего пользователя
 */
export async function updateUserNameReal(name: string): Promise<User> {
  const idempotencyKey = generateIdempotencyKey('user-name');
  await api.patch('/api/user/name', {
    name: name.trim(),
  }, {
    headers: { 'Idempotency-Key': idempotencyKey },
  });
  return getCurrentUserReal();
}

/**
 * DELETE /api/user/me
 * Выполняет мягкое удаление профиля текущего пользователя
 */
export async function deleteCurrentUserReal(): Promise<void> {
  await api.delete('/api/user/me');
  setStoredToken(null);
  setStoredRefreshId(null);
}

/**
 * PUT /api/user/me/avatar
 * Загружает и устанавливает квадратный аватар пользователя (multipart/form-data)
 */
export async function updateAvatarReal(file: File | Blob): Promise<{ image: string }> {
  const formData = new FormData();
  formData.append('file', file);
  return api.put<{ image: string }>('/api/user/me/avatar', formData);
}

/**
 * GET /api/user/me/freeCoin
 * Начисляет 1000 бесплатных монет на баланс авторизованного пользователя
 */
export async function getFreeCoinReal(): Promise<User> {
  const idempotencyKey = generateIdempotencyKey('free-coin');
  const user = await api.get<User>('/api/user/me/freeCoin', {
    headers: {
      'Idempotency-Key': idempotencyKey,
    },
  });
  return {
    ...user,
    balance: Number(user.balance ?? 0),
    hold: Number(user.hold ?? 0),
  };
}
