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

interface RequestOptions extends RequestInit {
  params?: Record<string, string | number | boolean | undefined>;
  skipAuth?: boolean;
}

class ApiClient {
  private isRefreshing = false;
  private refreshSubscribers: Array<(token: string) => void> = [];

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

    const storedToken = getStoredToken();
    if (!skipAuth && storedToken) {
      reqHeaders['Authorization'] = `Bearer ${storedToken}`;
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
      const serverErrMsg = jsonBody?.error || response.statusText || 'Ошибка сервера';
      throw new ApiError('http_error', serverErrMsg, response.status, jsonBody);
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
    return this.request<T>(url, { ...options, method: 'GET' });
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

export function generateIdempotencyKey(prefix = 'idemp'): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return `${prefix}-${crypto.randomUUID()}`;
  }
  return `${prefix}-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
}
