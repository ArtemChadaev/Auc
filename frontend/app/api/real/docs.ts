import { api, generateIdempotencyKey } from '~/api/client';
import type { Document, UserConsent } from '~/api/types';

/**
 * GET /api/document/
 * Возвращает список активных нормативных документов
 */
export async function listDocumentsReal(): Promise<Document[]> {
  const docs = await api.get<Document[]>('/api/document/');
  return Array.isArray(docs) ? docs : [];
}

/**
 * GET /api/document/name/{name} или GET /api/document/{id}
 * Возвращает актуальную версию документа по имени или ID
 */
export async function getDocumentReal(nameOrId: string | number): Promise<Document> {
  if (typeof nameOrId === 'number' || /^\d+$/.test(String(nameOrId))) {
    return api.get<Document>(`/api/document/${nameOrId}`);
  }

  try {
    return await api.get<Document>(`/api/document/name/${encodeURIComponent(nameOrId)}`);
  } catch (err: any) {
    // Если 404, пробуем найти в общем списке документов
    const all = await listDocumentsReal().catch(() => []);
    const found = all.find(
      (d) => d.name.toLowerCase() === String(nameOrId).toLowerCase()
    );
    if (found) return found;
    throw err;
  }
}

/**
 * POST /api/document/consents
 * Фиксирует согласие текущего пользователя с документом
 */
export async function recordConsentReal(documentId: number): Promise<UserConsent> {
  const idempotencyKey = generateIdempotencyKey(`consent-${documentId}`);
  return api.post<UserConsent>('/api/document/consents', {
    document_id: Number(documentId),
  }, {
    headers: { 'Idempotency-Key': idempotencyKey },
  });
}

/**
 * GET /api/document/consents
 * Возвращает все согласия с документами для текущего авторизованного пользователя
 */
export async function getUserConsentsReal(): Promise<UserConsent[]> {
  const consents = await api.get<UserConsent[]>('/api/document/consents');
  return Array.isArray(consents) ? consents : [];
}

/**
 * GET /api/document/consents/{id}
 * Проверяет и возвращает согласие текущего пользователя с конкретным документом
 */
export async function getConsentByDocumentIdReal(documentId: number): Promise<UserConsent> {
  return api.get<UserConsent>(`/api/document/consents/${documentId}`);
}
