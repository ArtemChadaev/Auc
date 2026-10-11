import { mockDb } from './mocks/db';
import { simulateLatency } from './mocks/latency';
import { listDocumentsReal, getDocumentReal, recordConsentReal } from './real/docs';
import type { Document } from './types';

const USE_MOCKS = true;

export async function listDocuments(): Promise<Document[]> {
  if (USE_MOCKS) {
    await simulateLatency();
    return mockDb.listDocuments();
  }
  return listDocumentsReal();
}

export async function getDocument(name: string): Promise<Document> {
  if (USE_MOCKS) {
    await simulateLatency();
    return mockDb.getDocument(name);
  }
  return getDocumentReal(name);
}

export async function recordConsent(documentId: number): Promise<void> {
  if (USE_MOCKS) {
    await simulateLatency();
    mockDb.recordConsent(documentId);
    return;
  }
  return recordConsentReal(documentId);
}
