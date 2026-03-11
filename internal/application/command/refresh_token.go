package command

import (
	"context"
	"fmt"
	"log/slog"

	authrepo "github.com/atls-academy/engineer-challenge/internal/domain/auth/repository"
	"github.com/atls-academy/engineer-challenge/internal/infrastructure/crypto"
	"github.com/atls-academy/engineer-challenge/internal/infrastructure/eventbus"
)

type RefreshTokenCommand struct {
	RefreshToken string
}

type RefreshTokenResult struct {
	AccessToken  string
	RefreshToken string
}

type RefreshTokenHandler struct {
	sessionRepo authrepo.SessionRepository
	tokenIssuer crypto.TokenIssuer
	bus         eventbus.EventBus
	logger      *slog.Logger
}

func NewRefreshTokenHandler(sessionRepo authrepo.SessionRepository, tokenIssuer crypto.TokenIssuer, bus eventbus.EventBus, logger *slog.Logger) *RefreshTokenHandler {
	return &RefreshTokenHandler{sessionRepo: sessionRepo, tokenIssuer: tokenIssuer, bus: bus, logger: logger}
}

func (h *RefreshTokenHandler) Handle(ctx context.Context, cmd RefreshTokenCommand) (*RefreshTokenResult, error) {
	session, err := h.sessionRepo.FindByRefreshToken(ctx, cmd.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	if !session.IsActive() {
		return nil, fmt.Errorf("session expired or revoked")
	}

	newAccessToken, err := h.tokenIssuer.IssueAccessToken(session.UserID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to issue access token: %w", err)
	}

	newRefreshToken, err := h.tokenIssuer.IssueRefreshToken(session.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to issue refresh token: %w", err)
	}

	session.RotateRefreshToken(newRefreshToken)

	if err := h.sessionRepo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	h.logger.Info("token refreshed", "user_id", session.UserID, "session_id", session.ID.String())
	return &RefreshTokenResult{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}
