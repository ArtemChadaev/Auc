import { ApiError } from './errors';

const TOKEN_KEY = 'auc_access_token';
const REFRESH_ID_KEY = 'auc_refresh_id';

export function getStoredToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem(TOKEN_KEY);
}

export function setStoredToken(token: string | null): void {
  if (typeof window === 'undefined') return;
  if (token) {
    localStorage.setItem(TOKEN_KEY, token);
  } else {
    localStorage.removeItem(TOKEN_KEY);
  }
}

export function getStoredRefreshId(): number | null {
  if (typeof window === 'undefined') return null;
  const val = localStorage.getItem(REFRESH_ID_KEY);
  return val ? parseInt(val, 10) : null;
}

export function setStoredRefreshId(id: number | null): void {
  if (typeof window === 'undefined') return;
  if (id !== null && id !== undefined) {
    localStorage.setItem(REFRESH_ID_KEY, String(id));
  } else {
    localStorage.removeItem(REFRESH_ID_KEY);
  }
}

interface IdempotencyCacheEntry {
  key: string;
  expiresAt: number;
}

const IDEMPOTENCY_TTL_MS = 8000; // 8 секунд (в диапазоне 5-10 секунд)
const idempotencyKeyMap = new Map<string, IdempotencyCacheEntry>();

function simpleHash(str: string): string {
  let hash = 0x811c9dc5;
  for (let i = 0; i < str.length; i++) {
    hash ^= str.charCodeAt(i);
    hash += (hash << 1) + (hash << 4) + (hash << 7) + (hash << 8) + (hash << 24);
  }
  return (hash >>> 0).toString(16);
}

export function getOrGenerateIdempotencyKey(
  method: string,
  endpoint: string,
  body?: any,
  providedKey?: string
): string {
  if (providedKey) return providedKey;

  let bodyStr = '';
  if (body) {
    if (typeof body === 'string') {
      bodyStr = body;
    } else if (typeof FormData !== 'undefined' && body instanceof FormData) {
      bodyStr = '[FormData]';
    } else {
      try {
        bodyStr = JSON.stringify(body);
      } catch {
        bodyStr = String(body);
      }
    }
  }

  const sig = `${method.toUpperCase()}:${endpoint}:${bodyStr}`;
  const now = Date.now();

  // Очистка устаревших записей при разрастании кеша
  if (idempotencyKeyMap.size > 200) {
    for (const [k, v] of idempotencyKeyMap.entries()) {
      if (v.expiresAt <= now) {
        idempotencyKeyMap.delete(k);
      }
    }
  }

  const existing = idempotencyKeyMap.get(sig);
  if (existing && existing.expiresAt > now) {
    return existing.key;
  }

  const hash = simpleHash(sig);
  const randomSuffix =
    typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
      ? crypto.randomUUID().slice(0, 8)
      : Math.random().toString(36).substring(2, 8);
  const key = `idemp-${hash}-${now}-${randomSuffix}`;

  idempotencyKeyMap.set(sig, {
    key,
    expiresAt: now + IDEMPOTENCY_TTL_MS,
  });

  return key;
}

export function generateIdempotencyKey(prefix = 'idemp'): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return `${prefix}-${crypto.randomUUID()}`;
  }
  return `${prefix}-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
}

interface RequestOptions extends RequestInit {
  params?: Record<string, string | number | boolean | undefined>;
  skipAuth?: boolean;
}

class ApiClient {
  private isRefreshing = false;
  private refreshSubscribers: Array<(token: string) => void> = [];
  private inFlightGets = new Map<string, Promise<any>>();

  private onTokenRefreshed(token: string) {
    this.refreshSubscribers.forEach((cb) => cb(token));
    this.refreshSubscribers = [];
  }

  private addRefreshSubscriber(cb: (token: string) => void) {
    this.refreshSubscribers.push(cb);
  }

  public async request<T = any>(endpoint: string, options: RequestOptions = {}): Promise<T> {
    const { params, skipAuth, headers = {}, ...customConfig } = options;

    let url = endpoint.startsWith('http') ? endpoint : endpoint;
    if (params) {
      const searchParams = new URLSearchParams();
      Object.entries(params).forEach(([key, val]) => {
        if (val !== undefined && val !== null && val !== '') {
          searchParams.append(key, String(val));
        }
      });
      const qs = searchParams.toString();
      if (qs) {
        url += (url.includes('?') ? '&' : '?') + qs;
      }
    }

    const reqHeaders: Record<string, string> = {
      Accept: 'application/json',
      ...(headers as Record<string, string>),
    };

    const method = (customConfig.method || 'GET').toUpperCase();
    const storedToken = getStoredToken();
    if (!skipAuth && storedToken) {
      reqHeaders['Authorization'] = `Bearer ${storedToken}`;

      // Автоматическое добавление Idempotency-Key с TTL 5-10 сек для одинаковых запросов
      const isMutating = ['POST', 'PUT', 'PATCH', 'DELETE'].includes(method) || url.includes('/freeCoin');
      if (isMutating && !reqHeaders['Idempotency-Key']) {
        reqHeaders['Idempotency-Key'] = getOrGenerateIdempotencyKey(
          method,
          url,
          customConfig.body
        );
      }
    }

    const config: RequestInit = {
      ...customConfig,
      headers: reqHeaders,
      credentials: 'include', // send refresh_token cookie
    };

    let response: Response;
    try {
      response = await fetch(url, config);
    } catch (networkError: any) {
      throw new ApiError('network_error', networkError?.message || 'Сетевая ошибка', 0);
    }

    // Обработка 401: если токен был, пытаемся обновить его через /api/auth/refresh
    if (response.status === 401 && !skipAuth && !url.includes('/api/auth/')) {
      if (!storedToken) {
        // Пользователь просто не авторизован (гость)
        throw new ApiError('unauthorized', 'Не авторизован', 401);
      }

      if (!this.isRefreshing) {
        this.isRefreshing = true;
        try {
          const refreshRes = await fetch('/api/auth/refresh', {
            method: 'POST',
            credentials: 'include',
            headers: { Accept: 'application/json' },
          });

          if (refreshRes.ok) {
            const refreshData = await refreshRes.json();
            const newToken = refreshData?.data?.access_token;
            if (newToken) {
              setStoredToken(newToken);
              this.onTokenRefreshed(newToken);
              this.isRefreshing = false;

              // Повторяем запрос с новым токеном
              reqHeaders['Authorization'] = `Bearer ${newToken}`;
              return this.request<T>(endpoint, {
                ...options,
                headers: reqHeaders,
              });
            }
          }
        } catch {
          // Игнорируем ошибку рефреша
        } finally {
          this.isRefreshing = false;
        }

        // Если рефреш не удался — сбрасываем токен
        setStoredToken(null);
        setStoredRefreshId(null);
      } else {
        // Ожидаем завершения параллельного рефреша
        return new Promise<T>((resolve, reject) => {
          this.addRefreshSubscriber(async (newToken: string) => {
            try {
              reqHeaders['Authorization'] = `Bearer ${newToken}`;
              const retryRes = await this.request<T>(endpoint, {
                ...options,
                headers: reqHeaders,
              });
              resolve(retryRes);
            } catch (err) {
              reject(err);
            }
          });
        });
      }
    }

    // Читаем тело ответа
    let jsonBody: any = null;
    const contentType = response.headers.get('content-type') || '';
    if (contentType.includes('application/json')) {
      try {
        jsonBody = await response.json();
      } catch {
        jsonBody = null;
      }
    }

    if (!response.ok) {
      let serverErrMsg = jsonBody?.error || response.statusText || 'Ошибка сервера';
      let errorCode = 'http_error';

      // Специализированная валидация и обработка HTTP 429 Too Many Requests
      if (response.status === 429) {
        errorCode = 'rate_limit';
        const lower = serverErrMsg.toLowerCase();
        if (lower.includes('free coin') || lower.includes('once per day') || lower.includes('freecoin') || lower.includes('бонус')) {
          errorCode = 'free_coin_limit';
          serverErrMsg = 'Бесплатные монеты (+1 000 🪙) можно получить только один раз в сутки (сброс в 00:00 UTC).';
        } else if (lower.includes('photo upload') || lower.includes('фото')) {
          errorCode = 'photo_upload_limit';
          serverErrMsg = 'Превышен лимит загрузки фотографий (максимум 30 фото в час). Пожалуйста, подождите немного.';
        } else if (lower.includes('download') || lower.includes('скачиван')) {
          errorCode = 'download_limit';
          serverErrMsg = 'Превышен суточный лимит скачиваний (максимум 1 скачивание в сутки).';
        } else if (lower.includes('bid') || lower.includes('ставк')) {
          errorCode = 'bid_rate_limit';
          serverErrMsg = 'Слишком частые ставки. Пожалуйста, подождите пару секунд.';
        } else if (lower.includes('endpoint rate limit') || lower.includes('global rate limit') || lower.includes('rate limit')) {
          errorCode = 'rate_limit';
          serverErrMsg = 'Слишком много запросов (429). Пожалуйста, подождите 1 минуту перед повторной попыткой.';
        } else {
          errorCode = 'rate_limit';
          serverErrMsg = 'Превышен лимит запросов (429 Too Many Requests). Пожалуйста, подождите немного перед повторной попыткой.';
        }
      }

      throw new ApiError(errorCode, serverErrMsg, response.status, {
        ...(jsonBody || {}),
        retryAfter: response.headers.get('Retry-After'),
      });
    }

    if (jsonBody && typeof jsonBody === 'object' && 'error' in jsonBody && jsonBody.error) {
      throw new ApiError('server_error', jsonBody.error, response.status, jsonBody);
    }

    // Если ответ обёрнут в httpx.Response { data: ... }
    if (jsonBody && typeof jsonBody === 'object' && 'data' in jsonBody) {
      return jsonBody.data as T;
    }

    return (jsonBody ?? null) as T;
  }

  public get<T = any>(url: string, options?: RequestOptions): Promise<T> {
    const cacheKey = url + (options?.params ? JSON.stringify(options.params) : '');
    const inFlight = this.inFlightGets.get(cacheKey);
    if (inFlight) {
      return inFlight as Promise<T>;
    }
    const p = this.request<T>(url, { ...options, method: 'GET' }).finally(() => {
      this.inFlightGets.delete(cacheKey);
    });
    this.inFlightGets.set(cacheKey, p);
    return p;
  }

  public post<T = any>(url: string, body?: any, options?: RequestOptions): Promise<T> {
    const isFormData = typeof FormData !== 'undefined' && body instanceof FormData;
    const headers = { ...(options?.headers || {}) } as Record<string, string>;
    if (!isFormData && body !== undefined) {
      headers['Content-Type'] = 'application/json';
    }

    return this.request<T>(url, {
      ...options,
      method: 'POST',
      headers,
      body: isFormData ? body : body !== undefined ? JSON.stringify(body) : undefined,
    });
  }

  public put<T = any>(url: string, body?: any, options?: RequestOptions): Promise<T> {
    const isFormData = typeof FormData !== 'undefined' && body instanceof FormData;
    const headers = { ...(options?.headers || {}) } as Record<string, string>;
    if (!isFormData && body !== undefined) {
      headers['Content-Type'] = 'application/json';
    }

    return this.request<T>(url, {
      ...options,
      method: 'PUT',
      headers,
      body: isFormData ? body : body !== undefined ? JSON.stringify(body) : undefined,
    });
  }

  public patch<T = any>(url: string, body?: any, options?: RequestOptions): Promise<T> {
    const isFormData = typeof FormData !== 'undefined' && body instanceof FormData;
    const headers = { ...(options?.headers || {}) } as Record<string, string>;
    if (!isFormData && body !== undefined) {
      headers['Content-Type'] = 'application/json';
    }

    return this.request<T>(url, {
      ...options,
      method: 'PATCH',
      headers,
      body: isFormData ? body : body !== undefined ? JSON.stringify(body) : undefined,
    });
  }

  public delete<T = any>(url: string, options?: RequestOptions): Promise<T> {
    return this.request<T>(url, { ...options, method: 'DELETE' });
  }
}

export const api = new ApiClient();
