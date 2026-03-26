package usecase

import (
	"context"
	"errors"
	"log/slog"

	"github.com/skiphead/go-letopis/internal/domain/entity"
	"github.com/skiphead/go-letopis/internal/domain/repository"
)

// UserUseCase defines the interface for user-related operations.
type UserUseCase interface {
	Start(ctx context.Context, user *entity.User) (*entity.User, error)
	Validate(ctx context.Context, userID int64) bool
}

// userUseCase implements UserUseCase.
type userUseCase struct {
	userRepo repository.UserRepository
	logger   *slog.Logger
}

// NewUserUseCase creates a new UserUseCase instance.
func NewUserUseCase(userRepo repository.UserRepository, logger *slog.Logger) UserUseCase {
	return &userUseCase{
		userRepo: userRepo,
		logger:   logger,
	}
}

// Start initializes a new user session.
// If the user already exists, it returns an error.
func (uc *userUseCase) Start(ctx context.Context, user *entity.User) (*entity.User, error) {
	result, err := uc.userRepo.Get(ctx, user.TelegramID)
	if err != nil {
		return nil, err
	}
	if result != nil {
		return nil, errors.New("user already started")
	}

	createResult, err := uc.userRepo.Create(ctx, &entity.User{
		TelegramID: user.TelegramID,
		UserName:   user.UserName,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
	})
	if err != nil {
		return nil, err
	}

	return createResult, nil
}

// Validate checks if a user exists and is valid.
func (uc *userUseCase) Validate(ctx context.Context, userID int64) bool {
	result, err := uc.userRepo.Get(ctx, userID)
	if err != nil {
		return false
	}
	if result != nil {
		return result.TelegramID == userID
	}
	return false
}
