import { mockDb } from './mocks/db';
import { simulateLatency } from './mocks/latency';
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

const USE_MOCKS = true;

/**
 * Шаг 1: POST /api/upload/url
 * Запрашивает Presigned PUT URL и ключ загрузки
 */
export async function getUploadUrl(req: UploadUrlRequest): Promise<UploadUrlResponse> {
  if (USE_MOCKS) {
    await simulateLatency(150);
    return mockDb.createUploadUrl(req);
  }
  return getUploadUrlReal(req);
}

/**
 * Шаг 2: Прямой PUT бинарных данных файла в S3 по presigned URL
 * В моке плавно анимирует прогресс 0-100%
 */
export async function uploadBinaryToS3(
  url: string,
  file: File | Blob,
  onProgress?: (percent: number) => void
): Promise<void> {
  if (USE_MOCKS) {
    // Симуляция стриминговой заливки в S3
    const totalSteps = 10;
    for (let step = 1; step <= totalSteps; step++) {
      await new Promise((r) => setTimeout(r, 60));
      if (onProgress) {
        onProgress(Math.round((step / totalSteps) * 100));
      }
    }
    return;
  }
  return uploadBinaryToS3Real(url, file, onProgress);
}

/**
 * Шаг 3: POST /api/upload/confirm (Заголовок X-Upload-Key: <key>)
 * Проверяет «каплю» (размер, первые 512 байт, mime-тип) и переводит в статус 'processing'
 */
export async function confirmUpload(key: string): Promise<ConfirmUploadResponse> {
  if (USE_MOCKS) {
    await simulateLatency(200);
    return mockDb.confirmUpload(key);
  }
  return confirmUploadReal(key);
}

/**
 * Шаг 4: GET /api/upload/status?key=<uuid>
 * Опрашивает статус обработки файла воркером (воркер-пул N=5)
 */
export async function getUploadStatus(key: string): Promise<UploadStatusResponse> {
  if (USE_MOCKS) {
    await simulateLatency(100);
    return mockDb.getUploadStatus(key);
  }
  return getUploadStatusReal(key);
}

/**
 * POST /api/upload/cancel
 * Отменяет активную загрузку и освобождает слот
 */
export async function cancelUpload(key?: string): Promise<void> {
  if (USE_MOCKS) {
    await simulateLatency(100);
    return mockDb.cancelUpload(key);
  }
  return cancelUploadReal(key);
}
