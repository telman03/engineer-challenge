package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/atls-academy/engineer-challenge/internal/domain/auth/aggregate"
	"github.com/atls-academy/engineer-challenge/internal/domain/auth/repository"
	"github.com/atls-academy/engineer-challenge/internal/domain/auth/valueobject"
)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) repository.SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Save(ctx context.Context, session *aggregate.Session) error {
	query := `INSERT INTO sessions (id, user_id, refresh_token, user_agent, ip, expires_at, created_at)
	           VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query,
		session.ID.String(), session.UserID, session.RefreshToken,
		session.UserAgent, session.IP, session.ExpiresAt, session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert session: %w", err)
	}
	return nil
}

func (r *SessionRepository) FindByID(ctx context.Context, id valueobject.SessionID) (*aggregate.Session, error) {
	return r.findOne(ctx, "SELECT id, user_id, refresh_token, user_agent, ip, expires_at, created_at, revoked_at FROM sessions WHERE id = $1", id.String())
}

func (r *SessionRepository) FindByRefreshToken(ctx context.Context, token string) (*aggregate.Session, error) {
	return r.findOne(ctx, "SELECT id, user_id, refresh_token, user_agent, ip, expires_at, created_at, revoked_at FROM sessions WHERE refresh_token = $1", token)
}

func (r *SessionRepository) FindActiveByUserID(ctx context.Context, userID string) ([]*aggregate.Session, error) {
	query := `SELECT id, user_id, refresh_token, user_agent, ip, expires_at, created_at, revoked_at
	          FROM sessions WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
	          ORDER BY created_at ASC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*aggregate.Session
	for rows.Next() {
		s, err := r.scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *SessionRepository) Update(ctx context.Context, session *aggregate.Session) error {
	query := `UPDATE sessions SET refresh_token = $1, expires_at = $2, revoked_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query,
		session.RefreshToken, session.ExpiresAt, session.RevokedAt, session.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	return nil
}

func (r *SessionRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	query := `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke sessions: %w", err)
	}
	return nil
}

func (r *SessionRepository) findOne(ctx context.Context, query string, args ...interface{}) (*aggregate.Session, error) {
	row := r.db.QueryRowContext(ctx, query, args...)

	var (
		idStr, userID, refreshToken, userAgent, ip string
		expiresAt, createdAt                       time.Time
		revokedAt                                  *time.Time
	)

	if err := row.Scan(&idStr, &userID, &refreshToken, &userAgent, &ip, &expiresAt, &createdAt, &revokedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to scan session: %w", err)
	}

	return r.reconstruct(idStr, userID, refreshToken, userAgent, ip, expiresAt, createdAt, revokedAt)
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func (r *SessionRepository) scanSession(row scanner) (*aggregate.Session, error) {
	var (
		idStr, userID, refreshToken, userAgent, ip string
		expiresAt, createdAt                       time.Time
		revokedAt                                  *time.Time
	)
	if err := row.Scan(&idStr, &userID, &refreshToken, &userAgent, &ip, &expiresAt, &createdAt, &revokedAt); err != nil {
		return nil, fmt.Errorf("failed to scan session: %w", err)
	}
	return r.reconstruct(idStr, userID, refreshToken, userAgent, ip, expiresAt, createdAt, revokedAt)
}

func (r *SessionRepository) reconstruct(idStr, userID, refreshToken, userAgent, ip string, expiresAt, createdAt time.Time, revokedAt *time.Time) (*aggregate.Session, error) {
	id, err := valueobject.SessionIDFromString(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}
	return aggregate.ReconstructSession(id, userID, refreshToken, userAgent, ip, expiresAt, createdAt, revokedAt), nil
}
