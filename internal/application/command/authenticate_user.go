package command

import (
	"context"
	"fmt"
	"log/slog"

	authaggregate "github.com/atls-academy/engineer-challenge/internal/domain/auth/aggregate"
	authrepo "github.com/atls-academy/engineer-challenge/internal/domain/auth/repository"
	"github.com/atls-academy/engineer-challenge/internal/domain/identity/repository"
	"github.com/atls-academy/engineer-challenge/internal/domain/identity/valueobject"
	"github.com/atls-academy/engineer-challenge/internal/infrastructure/crypto"
	"github.com/atls-academy/engineer-challenge/internal/infrastructure/eventbus"
)

type AuthenticateUserCommand struct {
	Email     string
	Password  string
	UserAgent string
	IP        string
}

type AuthenticateUserResult struct {
	AccessToken  string
	RefreshToken string
	UserID       string
	Email        string
}

type AuthenticateUserHandler struct {
	userRepo    repository.UserRepository
	sessionRepo authrepo.SessionRepository
	hasher      crypto.PasswordHasher
	tokenIssuer crypto.TokenIssuer
	bus         eventbus.EventBus
	logger      *slog.Logger
}

func NewAuthenticateUserHandler(
	userRepo repository.UserRepository,
	sessionRepo authrepo.SessionRepository,
	hasher crypto.PasswordHasher,
	tokenIssuer crypto.TokenIssuer,
	bus eventbus.EventBus,
	logger *slog.Logger,
) *AuthenticateUserHandler {
	return &AuthenticateUserHandler{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		hasher:      hasher,
		tokenIssuer: tokenIssuer,
		bus:         bus,
		logger:      logger,
	}
}

func (h *AuthenticateUserHandler) Handle(ctx context.Context, cmd AuthenticateUserCommand) (*AuthenticateUserResult, error) {
	h.logger.Info("processing authentication", "email", cmd.Email)

	email, err := valueobject.NewEmail(cmd.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	user, err := h.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if !user.IsActive() {
		return nil, fmt.Errorf("account is not active")
	}

	if !h.hasher.Verify(cmd.Password, user.PasswordHash) {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Enforce max concurrent sessions
	activeSessions, err := h.sessionRepo.FindActiveByUserID(ctx, user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to check sessions: %w", err)
	}
	if len(activeSessions) >= authaggregate.MaxSessions {
		oldest := activeSessions[0]
		for _, s := range activeSessions[1:] {
			if s.CreatedAt.Before(oldest.CreatedAt) {
				oldest = s
			}
		}
		if err := oldest.Revoke(); err == nil {
			_ = h.sessionRepo.Update(ctx, oldest)
		}
	}

	accessToken, err := h.tokenIssuer.IssueAccessToken(user.ID.String(), user.Email.String())
	if err != nil {
		return nil, fmt.Errorf("failed to issue access token: %w", err)
	}

	refreshToken, err := h.tokenIssuer.IssueRefreshToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to issue refresh token: %w", err)
	}

	session, err := authaggregate.CreateSession(user.ID.String(), refreshToken, cmd.UserAgent, cmd.IP)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	if err := h.sessionRepo.Save(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	for _, evt := range session.Events() {
		h.bus.Publish(ctx, evt)
	}

	h.logger.Info("user authenticated", "user_id", user.ID.String())
	return &AuthenticateUserResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		UserID:       user.ID.String(),
		Email:        user.Email.String(),
	}, nil
}
