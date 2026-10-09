package documents

import (
	"context"
	"fmt"
	"strings"
	"uuid"
)

type repo interface {
	getActiveDocuments(ctx context.Context) ([]Document, error)
	getDocumentByID(ctx context.Context, id int64) (Document, error)
	getActiveDocumentByName(ctx context.Context, name string) (Document, error)

	createConsent(ctx context.Context, docID int64, uid uuid.UUID) (UserConsent, error)
	getUserConsents(ctx context.Context, uid uuid.UUID) ([]UserConsent, error)
	getConsentByDocumentID(ctx context.Context, docID int64, uid uuid.UUID) (UserConsent, error)
}

type Service struct {
	repo repo
}

func NewService(repo repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetActiveDocuments(ctx context.Context) ([]Document, error) {
	docs, err := s.repo.getActiveDocuments(ctx)
	if err != nil {
		return nil, fmt.Errorf("documentService.GetActiveDocuments: %w", err)
	}
	return docs, nil
}

func (s *Service) GetDocumentByID(ctx context.Context, id int64) (Document, error) {
	if id <= 0 {
		return Document{}, fmt.Errorf("documentService.GetDocumentByID: %w", ErrInvalidID)
	}
	doc, err := s.repo.getDocumentByID(ctx, id)
	if err != nil {
		return Document{}, fmt.Errorf("documentService.GetDocumentByID: %w", err)
	}
	return doc, nil
}

func (s *Service) GetActiveDocumentByName(ctx context.Context, name string) (Document, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return Document{}, fmt.Errorf("documentService.GetActiveDocumentByName: %w", ErrDocumentNotFound)
	}
	doc, err := s.repo.getActiveDocumentByName(ctx, trimmed)
	if err != nil {
		return Document{}, fmt.Errorf("documentService.GetActiveDocumentByName: %w", err)
	}
	return doc, nil
}

func (s *Service) CreateConsent(ctx context.Context, uid uuid.UUID, docID int64) (UserConsent, error) {
	if docID <= 0 {
		return UserConsent{}, fmt.Errorf("documentService.CreateConsent: %w", ErrMissingDocumentID)
	}

	// Verify document exists
	if _, err := s.repo.getDocumentByID(ctx, docID); err != nil {
		return UserConsent{}, fmt.Errorf("documentService.CreateConsent.checkDoc: %w", err)
	}

	consent, err := s.repo.createConsent(ctx, docID, uid)
	if err != nil {
		return UserConsent{}, fmt.Errorf("documentService.CreateConsent: %w", err)
	}
	return consent, nil
}

func (s *Service) GetUserConsents(ctx context.Context, uid uuid.UUID) ([]UserConsent, error) {
	consents, err := s.repo.getUserConsents(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("documentService.GetUserConsents: %w", err)
	}
	return consents, nil
}

func (s *Service) GetConsentByDocumentID(ctx context.Context, docID int64, uid uuid.UUID) (UserConsent, error) {
	if docID <= 0 {
		return UserConsent{}, fmt.Errorf("documentService.GetConsentByDocumentID: %w", ErrInvalidID)
	}
	consent, err := s.repo.getConsentByDocumentID(ctx, docID, uid)
	if err != nil {
		return UserConsent{}, fmt.Errorf("documentService.GetConsentByDocumentID: %w", err)
	}
	return consent, nil
}
