package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/atls-academy/engineer-challenge/internal/domain/identity/aggregate"
	"github.com/atls-academy/engineer-challenge/internal/domain/identity/repository"
	"github.com/atls-academy/engineer-challenge/internal/domain/identity/valueobject"
	"github.com/atls-academy/engineer-challenge/internal/infrastructure/crypto"
	"github.com/atls-academy/engineer-challenge/internal/infrastructure/eventbus"
)

type RegisterUserCommand struct {
	Email    string
	Password string
}

type RegisterUserResult struct {
	UserID string
	Email  string
}

type RegisterUserHandler struct {
	repo   repository.UserRepository
	hasher crypto.PasswordHasher
	bus    eventbus.EventBus
	logger *slog.Logger
}

func NewRegisterUserHandler(repo repository.UserRepository, hasher crypto.PasswordHasher, bus eventbus.EventBus, logger *slog.Logger) *RegisterUserHandler {
	return &RegisterUserHandler{repo: repo, hasher: hasher, bus: bus, logger: logger}
}

func (h *RegisterUserHandler) Handle(ctx context.Context, cmd RegisterUserCommand) (*RegisterUserResult, error) {
	h.logger.Info("processing registration", "email", cmd.Email)

	email, err := valueobject.NewEmail(cmd.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	_, err = valueobject.NewPassword(cmd.Password, valueobject.DefaultPasswordPolicy)
	if err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	exists, err := h.repo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email uniqueness: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("email already registered")
	}

	hash, err := h.hasher.Hash(cmd.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := aggregate.Register(email, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if err := h.repo.Save(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	for _, evt := range user.Events() {
		h.bus.Publish(ctx, evt)
	}

	h.logger.Info("user registered", "user_id", user.ID.String(), "email", cmd.Email)
	return &RegisterUserResult{UserID: user.ID.String(), Email: email.String()}, nil
}
