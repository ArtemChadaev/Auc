import { api, generateIdempotencyKey } from '~/api/client';
import type {
  UploadUrlRequest,
  UploadUrlResponse,
  ConfirmUploadResponse,
  UploadStatusResponse,
} from '~/api/types';

/**
 * POST /api/upload/url
 * Генерирует Presigned PUT URL для прямой загрузки файла в S3
 */
export async function getUploadUrlReal(req: UploadUrlRequest): Promise<UploadUrlResponse> {
  const idempotencyKey = `upload-${req.sha_256_hex}`;
  return api.post<UploadUrlResponse>('/api/upload/url', {
    file_size: Number(req.file_size),
    filename: req.filename,
    sha_256_hex: req.sha_256_hex,
  }, {
    headers: { 'Idempotency-Key': idempotencyKey },
  });
}

/**
 * PUT <presigned-url>
 * Прямой PUT бинарных данных файла в S3 с отслеживанием прогресса
 */
export async function uploadBinaryToS3Real(
  url: string,
  file: File | Blob,
  onProgress?: (percent: number) => void
): Promise<void> {
  // Важно: в Beget S3 presigned URL подписаны только content-length и host (X-Amz-SignedHeaders=content-length;host).
  // Если браузер передает заголовок Content-Type (включая непустой file.type), браузер инициирует
  // CORS-preflight OPTIONS запрос, который S3 отклоняет со статусом 403.
  // Передаем untyped Blob (file.slice без MIME типа) и не устанавливаем заголовок Content-Type,
  // в точности как в apiTest/item/item.http: PUT {{upload_url}} с чистым телом файла.
  const untypedBlob = file instanceof Blob ? file.slice(0, file.size, '') : file;

  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open('PUT', url, true);

    if (xhr.upload && onProgress) {
      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable && e.total > 0) {
          const percent = Math.min(99, Math.round((e.loaded / e.total) * 100));
          onProgress(percent);
        }
      };
    }

    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        if (onProgress) onProgress(100);
        resolve();
      } else {
        reject(new Error(`S3 upload failed with status ${xhr.status}: ${xhr.statusText}`));
      }
    };

    xhr.onerror = async () => {
      // Fallback на прямой fetch без заголовков
      try {
        const res = await fetch(url, {
          method: 'PUT',
          body: untypedBlob,
        });
        if (res.ok) {
          if (onProgress) onProgress(100);
          resolve();
          return;
        }
        reject(new Error(`S3 upload failed with status ${res.status}: ${res.statusText}`));
      } catch {
        reject(new Error('S3 upload network error'));
      }
    };

    xhr.send(untypedBlob);
  });
}

/**
 * POST /api/upload/confirm
 * Подтверждает успешную загрузку файла в S3 для проверки и старта фонового парсинга
 */
export async function confirmUploadReal(key: string): Promise<ConfirmUploadResponse> {
  return api.post<ConfirmUploadResponse>(
    '/api/upload/confirm',
    { key },
    {
      headers: {
        'X-Upload-Key': key,
        'Idempotency-Key': `confirm-${key}`,
      },
    }
  );
}

/**
 * GET /api/upload/status?key=<uuid>
 * Проверяет текущий статус обработки файла по ключу загрузки
 */
export async function getUploadStatusReal(key: string): Promise<UploadStatusResponse> {
  try {
    const res = await api.get<any>('/api/upload/status', {
      params: { key },
    });

    if (res && res.id) {
      // Статус 201 возвращает созданный Item
      return {
        status: 'completed',
        item: res,
        metadata: res.metadata,
      };
    }

    return {
      status: 'processing',
    };
  } catch (err: any) {
    if (err.status === 410) {
      return { status: 'cancelled' };
    }
    if (err.status === 404) {
      return { status: 'failed', error: 'Загрузка не найдена' };
    }
    throw err;
  }
}

/**
 * POST /api/upload/cancel
 * Отменяет активную загрузку пользователя, удаляет временный файл из S3 и освобождает слот
 */
export async function cancelUploadReal(key?: string): Promise<void> {
  const idempotencyKey = key ? `cancel-${key}` : generateIdempotencyKey('cancel-upload');
  await api.post('/api/upload/cancel', undefined, {
    headers: { 'Idempotency-Key': idempotencyKey },
  });
}
