package service

import (
	"context"
	"errors"

	"personae-tabula/internal/domain"
	"personae-tabula/internal/repository/postgres"
)

type UserService struct {
	userRepo *postgres.UserRepository
}

func NewUserService(userRepo *postgres.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) CreateUser(ctx context.Context, username, password, email string) (*domain.User, error) {
	existing, _ := s.userRepo.GetByUsername(ctx, username)
	if existing != nil {
		return nil, errors.New("username already taken")
	}

	user := &domain.User{
		Username: username,
		Email:    email,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	return s.userRepo.GetByUsername(ctx, username)
}
