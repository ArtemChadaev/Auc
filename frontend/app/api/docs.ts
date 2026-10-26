import {
  listDocumentsReal,
  getDocumentReal,
  recordConsentReal,
  getUserConsentsReal,
  getConsentByDocumentIdReal,
} from './real/docs';
import type { Document, UserConsent } from './types';

const USE_MOCKS = false;

export async function listDocuments(): Promise<Document[]> {
  return listDocumentsReal();
}

export async function getDocument(nameOrId: string | number): Promise<Document> {
  return getDocumentReal(nameOrId);
}

export async function recordConsent(documentId: number): Promise<UserConsent> {
  return recordConsentReal(documentId);
}

export async function getUserConsents(): Promise<UserConsent[]> {
  return getUserConsentsReal();
}

export async function getConsentByDocumentId(documentId: number): Promise<UserConsent> {
  return getConsentByDocumentIdReal(documentId);
}
