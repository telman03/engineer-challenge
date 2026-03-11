package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/atls-academy/engineer-challenge/internal/domain/identity/repository"
	"github.com/atls-academy/engineer-challenge/internal/domain/identity/valueobject"
	"github.com/atls-academy/engineer-challenge/internal/infrastructure/crypto"
	"github.com/atls-academy/engineer-challenge/internal/infrastructure/eventbus"
)

type ResetPasswordCommand struct {
	Email       string
	Token       string
	NewPassword string
}

type ResetPasswordHandler struct {
	repo   repository.UserRepository
	hasher crypto.PasswordHasher
	bus    eventbus.EventBus
	logger *slog.Logger
}

func NewResetPasswordHandler(repo repository.UserRepository, hasher crypto.PasswordHasher, bus eventbus.EventBus, logger *slog.Logger) *ResetPasswordHandler {
	return &ResetPasswordHandler{repo: repo, hasher: hasher, bus: bus, logger: logger}
}

func (h *ResetPasswordHandler) Handle(ctx context.Context, cmd ResetPasswordCommand) error {
	h.logger.Info("processing password reset")

	email, err := valueobject.NewEmail(cmd.Email)
	if err != nil {
		return fmt.Errorf("invalid email: %w", err)
	}

	_, err = valueobject.NewPassword(cmd.NewPassword, valueobject.DefaultPasswordPolicy)
	if err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}

	user, err := h.repo.FindByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("invalid reset request")
	}

	hash, err := h.hasher.Hash(cmd.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := user.ResetPassword(cmd.Token, hash); err != nil {
		return fmt.Errorf("password reset failed: %w", err)
	}

	if err := h.repo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	for _, evt := range user.Events() {
		h.bus.Publish(ctx, evt)
	}

	h.logger.Info("password reset completed", "user_id", user.ID.String())
	return nil
}
