package query

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	authrepo "github.com/atls-academy/engineer-challenge/internal/domain/auth/repository"
)

type SessionDTO struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	UserAgent string     `json:"user_agent"`
	IP        string     `json:"ip"`
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	IsActive  bool       `json:"is_active"`
}

type GetUserSessionsQuery struct {
	UserID string
}

type GetUserSessionsHandler struct {
	repo   authrepo.SessionRepository
	logger *slog.Logger
}

func NewGetUserSessionsHandler(repo authrepo.SessionRepository, logger *slog.Logger) *GetUserSessionsHandler {
	return &GetUserSessionsHandler{repo: repo, logger: logger}
}

func (h *GetUserSessionsHandler) Handle(ctx context.Context, q GetUserSessionsQuery) ([]SessionDTO, error) {
	if q.UserID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	sessions, err := h.repo.FindActiveByUserID(ctx, q.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sessions: %w", err)
	}

	dtos := make([]SessionDTO, 0, len(sessions))
	for _, s := range sessions {
		dtos = append(dtos, SessionDTO{
			ID:        s.ID.String(),
			UserID:    s.UserID,
			UserAgent: s.UserAgent,
			IP:        s.IP,
			ExpiresAt: s.ExpiresAt,
			CreatedAt: s.CreatedAt,
			RevokedAt: s.RevokedAt,
			IsActive:  s.IsActive(),
		})
	}
	return dtos, nil
}
