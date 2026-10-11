import type { Document } from '~/api/types';

// TODO(real-api): GET /api/document/
export async function listDocumentsReal(): Promise<Document[]> {
  throw new Error('Real API: listDocuments is not implemented');
}

// TODO(real-api): GET /api/document/name/{name}
export async function getDocumentReal(_name: string): Promise<Document> {
  throw new Error('Real API: getDocument is not implemented');
}

// TODO(real-api): POST /api/document/consents
export async function recordConsentReal(_documentId: number): Promise<void> {
  throw new Error('Real API: recordConsent is not implemented');
}
