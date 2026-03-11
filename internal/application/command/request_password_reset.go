package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/atls-academy/engineer-challenge/internal/domain/identity/repository"
	"github.com/atls-academy/engineer-challenge/internal/domain/identity/valueobject"
	"github.com/atls-academy/engineer-challenge/internal/infrastructure/eventbus"
)

type RequestPasswordResetCommand struct {
	Email string
}

type RequestPasswordResetResult struct {
	// Token is returned for the application layer to deliver via email.
	// In production, this would never be in an API response.
	Token string
}

type RequestPasswordResetHandler struct {
	repo   repository.UserRepository
	bus    eventbus.EventBus
	logger *slog.Logger
}

func NewRequestPasswordResetHandler(repo repository.UserRepository, bus eventbus.EventBus, logger *slog.Logger) *RequestPasswordResetHandler {
	return &RequestPasswordResetHandler{repo: repo, bus: bus, logger: logger}
}

func (h *RequestPasswordResetHandler) Handle(ctx context.Context, cmd RequestPasswordResetCommand) (*RequestPasswordResetResult, error) {
	h.logger.Info("processing password reset request", "email", cmd.Email)

	email, err := valueobject.NewEmail(cmd.Email)
	if err != nil {
		// Don't reveal validation details — always return success to prevent enumeration
		return &RequestPasswordResetResult{}, nil
	}

	user, err := h.repo.FindByEmail(ctx, email)
	if err != nil {
		h.logger.Info("password reset requested for unknown email", "email", cmd.Email)
		return &RequestPasswordResetResult{}, nil
	}

	token, err := user.RequestPasswordReset()
	if err != nil {
		return nil, fmt.Errorf("failed to generate reset token: %w", err)
	}

	if err := h.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	for _, evt := range user.Events() {
		h.bus.Publish(ctx, evt)
	}

	h.logger.Info("password reset token issued", "user_id", user.ID.String())
	return &RequestPasswordResetResult{Token: token.Token}, nil
}
