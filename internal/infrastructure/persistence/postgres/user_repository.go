package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/atls-academy/engineer-challenge/internal/domain/identity/aggregate"
	"github.com/atls-academy/engineer-challenge/internal/domain/identity/repository"
	"github.com/atls-academy/engineer-challenge/internal/domain/identity/valueobject"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) repository.UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Save(ctx context.Context, user *aggregate.User) error {
	query := `INSERT INTO users (id, email, password_hash, status, created_at, updated_at)
	           VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query,
		user.ID.String(), user.Email.String(), user.PasswordHash,
		string(user.Status), user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id valueobject.UserID) (*aggregate.User, error) {
	return r.findOne(ctx, "SELECT id, email, password_hash, status, reset_token, reset_token_expires_at, reset_token_used, created_at, updated_at FROM users WHERE id = $1", id.String())
}

func (r *UserRepository) FindByEmail(ctx context.Context, email valueobject.Email) (*aggregate.User, error) {
	return r.findOne(ctx, "SELECT id, email, password_hash, status, reset_token, reset_token_expires_at, reset_token_used, created_at, updated_at FROM users WHERE email = $1", email.String())
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email valueobject.Email) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email.String()).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}
	return exists, nil
}

func (r *UserRepository) Update(ctx context.Context, user *aggregate.User) error {
	var resetToken *string
	var resetTokenExpiresAt *time.Time
	var resetTokenUsed *bool
	if user.ResetToken != nil {
		resetToken = &user.ResetToken.Token
		resetTokenExpiresAt = &user.ResetToken.ExpiresAt
		resetTokenUsed = &user.ResetToken.Used
	}

	query := `UPDATE users SET email = $1, password_hash = $2, status = $3,
              reset_token = $4, reset_token_expires_at = $5, reset_token_used = $6,
              updated_at = $7 WHERE id = $8`
	_, err := r.db.ExecContext(ctx, query,
		user.Email.String(), user.PasswordHash, string(user.Status),
		resetToken, resetTokenExpiresAt, resetTokenUsed,
		user.UpdatedAt, user.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (r *UserRepository) findOne(ctx context.Context, query string, args ...interface{}) (*aggregate.User, error) {
	row := r.db.QueryRowContext(ctx, query, args...)

	var (
		idStr, emailStr, hash, status string
		resetToken                    *string
		resetTokenExpiresAt           *time.Time
		resetTokenUsed                *bool
		createdAt, updatedAt          time.Time
	)

	if err := row.Scan(&idStr, &emailStr, &hash, &status, &resetToken, &resetTokenExpiresAt, &resetTokenUsed, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	id, err := valueobject.UserIDFromString(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in database: %w", err)
	}
	email, err := valueobject.NewEmail(emailStr)
	if err != nil {
		return nil, fmt.Errorf("invalid email in database: %w", err)
	}

	var rt *valueobject.ResetToken
	if resetToken != nil && resetTokenExpiresAt != nil && resetTokenUsed != nil {
		rt = &valueobject.ResetToken{
			Token:     *resetToken,
			ExpiresAt: *resetTokenExpiresAt,
			Used:      *resetTokenUsed,
		}
	}

	return aggregate.Reconstruct(id, email, hash, aggregate.UserStatus(status), rt, createdAt, updatedAt), nil
}
