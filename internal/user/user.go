package user

import (
	"context"

	"golang.org/x/crypto/bcrypt"
)

type repo interface {
	Login(ctx context.Context, email string) (string, error)
	MiniRegister(ctx context.Context, name string, email string, password string) error
}

type Service struct {
	repo repo
}

func NewService(repo repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) Auth(ctx context.Context, email string, password string) error {
	passHash, err := s.repo.Login(ctx, email)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passHash), []byte(password)); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			// Потом добавить кастомные ошибки
			return err
		}
		return err
	}
	return nil
}

func (s *Service) Register(ctx context.Context, name string, email string, password string) error {
	pass := []byte(password)
	passHash, err := bcrypt.GenerateFromPassword(pass, bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.MiniRegister(ctx, name, email, string(passHash))
}
