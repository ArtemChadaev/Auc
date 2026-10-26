import {
  getUploadUrlReal,
  uploadBinaryToS3Real,
  confirmUploadReal,
  getUploadStatusReal,
  cancelUploadReal,
} from './real/upload';
import type {
  UploadUrlRequest,
  UploadUrlResponse,
  ConfirmUploadResponse,
  UploadStatusResponse,
} from './types';

const USE_MOCKS = false;

/**
 * Шаг 1: POST /api/upload/url
 * Запрашивает Presigned PUT URL и ключ загрузки
 */
export async function getUploadUrl(req: UploadUrlRequest): Promise<UploadUrlResponse> {
  return getUploadUrlReal(req);
}

/**
 * Шаг 2: Прямой PUT бинарных данных файла в S3 по presigned URL
 */
export async function uploadBinaryToS3(
  url: string,
  file: File | Blob,
  onProgress?: (percent: number) => void
): Promise<void> {
  return uploadBinaryToS3Real(url, file, onProgress);
}

/**
 * Шаг 3: POST /api/upload/confirm (Заголовок X-Upload-Key: <key>)
 * Проверяет «каплю» (размер, первые 512 байт, mime-тип) и переводит в статус 'processing'
 */
export async function confirmUpload(key: string): Promise<ConfirmUploadResponse> {
  return confirmUploadReal(key);
}

/**
 * Шаг 4: GET /api/upload/status?key=<uuid>
 * Опрашивает статус обработки файла воркером
 */
export async function getUploadStatus(key: string): Promise<UploadStatusResponse> {
  return getUploadStatusReal(key);
}

/**
 * POST /api/upload/cancel
 * Отменяет активную загрузку и освобождает слот
 */
export async function cancelUpload(key?: string): Promise<void> {
  return cancelUploadReal(key);
}
