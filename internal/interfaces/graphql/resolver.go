package graphql

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/atls-academy/engineer-challenge/internal/application/command"
	"github.com/atls-academy/engineer-challenge/internal/application/query"
	"github.com/atls-academy/engineer-challenge/internal/infrastructure/crypto"
	"github.com/atls-academy/engineer-challenge/internal/infrastructure/observability"
	"github.com/atls-academy/engineer-challenge/internal/infrastructure/persistence/redis"
	gql "github.com/graphql-go/graphql"
)

// Resolver holds all command and query handlers for GraphQL.
type Resolver struct {
	registerHandler      *command.RegisterUserHandler
	authenticateHandler  *command.AuthenticateUserHandler
	refreshTokenHandler  *command.RefreshTokenHandler
	requestResetHandler  *command.RequestPasswordResetHandler
	resetPasswordHandler *command.ResetPasswordHandler
	getUserHandler       *query.GetUserByIDHandler
	tokenValidator       crypto.TokenIssuer
	rateLimiter          *redis.RateLimiter
	logger               *slog.Logger
}

type ResolverConfig struct {
	RegisterHandler      *command.RegisterUserHandler
	AuthenticateHandler  *command.AuthenticateUserHandler
	RefreshTokenHandler  *command.RefreshTokenHandler
	RequestResetHandler  *command.RequestPasswordResetHandler
	ResetPasswordHandler *command.ResetPasswordHandler
	GetUserHandler       *query.GetUserByIDHandler
	TokenValidator       crypto.TokenIssuer
	RateLimiter          *redis.RateLimiter
	Logger               *slog.Logger
}

func NewResolver(cfg ResolverConfig) *Resolver {
	return &Resolver{
		registerHandler:      cfg.RegisterHandler,
		authenticateHandler:  cfg.AuthenticateHandler,
		refreshTokenHandler:  cfg.RefreshTokenHandler,
		requestResetHandler:  cfg.RequestResetHandler,
		resetPasswordHandler: cfg.ResetPasswordHandler,
		getUserHandler:       cfg.GetUserHandler,
		tokenValidator:       cfg.TokenValidator,
		rateLimiter:          cfg.RateLimiter,
		logger:               cfg.Logger,
	}
}

// Ensure context import is used.
var _ = context.Background

func (r *Resolver) Schema() (gql.Schema, error) {
	userType := gql.NewObject(gql.ObjectConfig{
		Name: "User",
		Fields: gql.Fields{
			"id":     &gql.Field{Type: gql.NewNonNull(gql.ID)},
			"email":  &gql.Field{Type: gql.NewNonNull(gql.String)},
			"status": &gql.Field{Type: gql.NewNonNull(gql.String)},
		},
	})

	authPayloadType := gql.NewObject(gql.ObjectConfig{
		Name: "AuthPayload",
		Fields: gql.Fields{
			"accessToken":  &gql.Field{Type: gql.NewNonNull(gql.String)},
			"refreshToken": &gql.Field{Type: gql.NewNonNull(gql.String)},
			"user":         &gql.Field{Type: gql.NewNonNull(userType)},
		},
	})

	tokenPairType := gql.NewObject(gql.ObjectConfig{
		Name: "TokenPair",
		Fields: gql.Fields{
			"accessToken":  &gql.Field{Type: gql.NewNonNull(gql.String)},
			"refreshToken": &gql.Field{Type: gql.NewNonNull(gql.String)},
		},
	})

	resetPayloadType := gql.NewObject(gql.ObjectConfig{
		Name: "ResetRequestPayload",
		Fields: gql.Fields{
			"success": &gql.Field{Type: gql.NewNonNull(gql.Boolean)},
			"token":   &gql.Field{Type: gql.String},
		},
	})

	queryType := gql.NewObject(gql.ObjectConfig{
		Name: "Query",
		Fields: gql.Fields{
			"me": &gql.Field{
				Type: gql.NewNonNull(userType),
				Resolve: func(p gql.ResolveParams) (interface{}, error) {
					token, ok := p.Context.Value(contextKeyToken).(string)
					if !ok || token == "" {
						return nil, fmt.Errorf("unauthorized")
					}
					claims, err := r.tokenValidator.ValidateAccessToken(token)
					if err != nil {
						return nil, fmt.Errorf("unauthorized")
					}
					user, err := r.getUserHandler.Handle(p.Context, query.GetUserByIDQuery{UserID: claims.UserID})
					if err != nil {
						return nil, err
					}
					return map[string]interface{}{
						"id":     user.ID,
						"email":  user.Email,
						"status": user.Status,
					}, nil
				},
			},
		},
	})

	mutationType := gql.NewObject(gql.ObjectConfig{
		Name: "Mutation",
		Fields: gql.Fields{
			"register": &gql.Field{
				Type: gql.NewNonNull(userType),
				Args: gql.FieldConfigArgument{
					"email":    &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
					"password": &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
				},
				Resolve: r.resolveRegister,
			},
			"login": &gql.Field{
				Type: gql.NewNonNull(authPayloadType),
				Args: gql.FieldConfigArgument{
					"email":    &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
					"password": &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
				},
				Resolve: r.resolveLogin,
			},
			"refreshToken": &gql.Field{
				Type: gql.NewNonNull(tokenPairType),
				Args: gql.FieldConfigArgument{
					"refreshToken": &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
				},
				Resolve: r.resolveRefreshToken,
			},
			"requestPasswordReset": &gql.Field{
				Type: gql.NewNonNull(resetPayloadType),
				Args: gql.FieldConfigArgument{
					"email": &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
				},
				Resolve: r.resolveRequestPasswordReset,
			},
			"resetPassword": &gql.Field{
				Type: gql.NewNonNull(gql.Boolean),
				Args: gql.FieldConfigArgument{
					"email":       &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
					"token":       &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
					"newPassword": &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
				},
				Resolve: r.resolveResetPassword,
			},
			"logout": &gql.Field{
				Type: gql.NewNonNull(gql.Boolean),
				Args: gql.FieldConfigArgument{
					"refreshToken": &gql.ArgumentConfig{Type: gql.NewNonNull(gql.String)},
				},
				Resolve: func(p gql.ResolveParams) (interface{}, error) {
					return true, nil
				},
			},
		},
	})

	return gql.NewSchema(gql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	})
}

func (r *Resolver) resolveRegister(p gql.ResolveParams) (interface{}, error) {
	start := time.Now()
	defer func() {
		observability.RequestDuration.WithLabelValues("register").Observe(time.Since(start).Seconds())
	}()

	email := p.Args["email"].(string)
	password := p.Args["password"].(string)

	if r.rateLimiter != nil {
		allowed, _ := r.rateLimiter.Allow(p.Context, "register:global", 20, time.Minute)
		if !allowed {
			observability.RateLimitHits.WithLabelValues("register").Inc()
			return nil, fmt.Errorf("too many registration attempts, please try again later")
		}
	}

	observability.RegistrationTotal.Inc()
	result, err := r.registerHandler.Handle(p.Context, command.RegisterUserCommand{
		Email:    email,
		Password: password,
	})
	if err != nil {
		observability.RegistrationErrors.Inc()
		return nil, err
	}

	return map[string]interface{}{
		"id":     result.UserID,
		"email":  result.Email,
		"status": "active",
	}, nil
}

func (r *Resolver) resolveLogin(p gql.ResolveParams) (interface{}, error) {
	start := time.Now()
	defer func() {
		observability.RequestDuration.WithLabelValues("login").Observe(time.Since(start).Seconds())
	}()

	email := p.Args["email"].(string)
	password := p.Args["password"].(string)

	ip, _ := p.Context.Value(contextKeyIP).(string)
	userAgent, _ := p.Context.Value(contextKeyUserAgent).(string)

	if r.rateLimiter != nil {
		allowed, _ := r.rateLimiter.Allow(p.Context, "login:"+ip, 10, time.Minute)
		if !allowed {
			observability.RateLimitHits.WithLabelValues("login").Inc()
			return nil, fmt.Errorf("too many login attempts, please try again later")
		}
	}

	observability.LoginTotal.Inc()
	result, err := r.authenticateHandler.Handle(p.Context, command.AuthenticateUserCommand{
		Email:     email,
		Password:  password,
		UserAgent: userAgent,
		IP:        ip,
	})
	if err != nil {
		observability.LoginErrors.Inc()
		return nil, err
	}

	return map[string]interface{}{
		"accessToken":  result.AccessToken,
		"refreshToken": result.RefreshToken,
		"user": map[string]interface{}{
			"id":     result.UserID,
			"email":  result.Email,
			"status": "active",
		},
	}, nil
}

func (r *Resolver) resolveRefreshToken(p gql.ResolveParams) (interface{}, error) {
	start := time.Now()
	defer func() {
		observability.RequestDuration.WithLabelValues("refresh_token").Observe(time.Since(start).Seconds())
	}()

	refreshToken := p.Args["refreshToken"].(string)

	observability.TokenRefreshTotal.Inc()
	result, err := r.refreshTokenHandler.Handle(p.Context, command.RefreshTokenCommand{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"accessToken":  result.AccessToken,
		"refreshToken": result.RefreshToken,
	}, nil
}

func (r *Resolver) resolveRequestPasswordReset(p gql.ResolveParams) (interface{}, error) {
	start := time.Now()
	defer func() {
		observability.RequestDuration.WithLabelValues("request_reset").Observe(time.Since(start).Seconds())
	}()

	email := p.Args["email"].(string)

	if r.rateLimiter != nil {
		allowed, _ := r.rateLimiter.Allow(p.Context, "reset:"+email, 3, 15*time.Minute)
		if !allowed {
			observability.RateLimitHits.WithLabelValues("reset").Inc()
			return nil, fmt.Errorf("too many reset attempts, please try again later")
		}
	}

	observability.PasswordResetRequestTotal.Inc()
	result, err := r.requestResetHandler.Handle(p.Context, command.RequestPasswordResetCommand{
		Email: email,
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"token":   result.Token,
	}, nil
}

func (r *Resolver) resolveResetPassword(p gql.ResolveParams) (interface{}, error) {
	start := time.Now()
	defer func() {
		observability.RequestDuration.WithLabelValues("reset_password").Observe(time.Since(start).Seconds())
	}()

	email := p.Args["email"].(string)
	token := p.Args["token"].(string)
	newPassword := p.Args["newPassword"].(string)

	err := r.resetPasswordHandler.Handle(p.Context, command.ResetPasswordCommand{
		Email:       email,
		Token:       token,
		NewPassword: newPassword,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}
