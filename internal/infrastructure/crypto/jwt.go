package crypto

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/atls-academy/engineer-challenge/internal/domain/auth/aggregate"
)

// JWTIssuer implements TokenIssuer using HMAC-SHA256 signed JWTs.
type JWTIssuer struct {
	secret []byte
	issuer string
}

func NewJWTIssuer(secret, issuer string) *JWTIssuer {
	return &JWTIssuer{
		secret: []byte(secret),
		issuer: issuer,
	}
}

func (j *JWTIssuer) IssueAccessToken(userID, email string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"iss":   j.issuer,
		"iat":   now.Unix(),
		"exp":   now.Add(aggregate.AccessTokenTTL).Unix(),
		"type":  "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(j.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}
	return signed, nil
}

func (j *JWTIssuer) IssueRefreshToken(userID string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  userID,
		"iss":  j.issuer,
		"iat":  now.Unix(),
		"exp":  now.Add(aggregate.RefreshTokenTTL).Unix(),
		"type": "refresh",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(j.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}
	return signed, nil
}

func (j *JWTIssuer) ValidateAccessToken(tokenStr string) (*TokenClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	tokenType, _ := claims["type"].(string)
	if tokenType != "access" {
		return nil, fmt.Errorf("not an access token")
	}

	userID, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)

	return &TokenClaims{UserID: userID, Email: email}, nil
}
