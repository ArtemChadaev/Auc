/**
 * Класс ошибок API и хелперы.
 * Не возвращает сырые серверные ошибки наружу, обеспечивает типизированную обработку.
 */

export type ErrorKind = 'race' | 'user' | 'auth' | 'system';

export class ApiError extends Error {
  public code: string;
  public status: number;
  public details?: unknown;

  constructor(code: string, message: string, status = 400, details?: unknown) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.status = status;
    this.details = details;
  }
}

export function isApiError(err: unknown): err is ApiError {
  return err instanceof ApiError;
}

export function toApiError(err: unknown): ApiError {
  if (err instanceof ApiError) {
    return err;
  }
  if (err instanceof Error) {
    return new ApiError('system_error', err.message, 500);
  }
  return new ApiError('unknown_error', 'Произошла непредвиденная ошибка', 500);
}
