package query

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/atls-academy/engineer-challenge/internal/domain/identity/repository"
	"github.com/atls-academy/engineer-challenge/internal/domain/identity/valueobject"
)

// UserDTO is the read-model projection of a User.
type UserDTO struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type GetUserByIDQuery struct {
	UserID string
}

type GetUserByIDHandler struct {
	repo   repository.UserRepository
	logger *slog.Logger
}

func NewGetUserByIDHandler(repo repository.UserRepository, logger *slog.Logger) *GetUserByIDHandler {
	return &GetUserByIDHandler{repo: repo, logger: logger}
}

func (h *GetUserByIDHandler) Handle(ctx context.Context, q GetUserByIDQuery) (*UserDTO, error) {
	id, err := valueobject.UserIDFromString(q.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	user, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return &UserDTO{
		ID:        user.ID.String(),
		Email:     user.Email.String(),
		Status:    string(user.Status),
		CreatedAt: user.CreatedAt,
	}, nil
}
