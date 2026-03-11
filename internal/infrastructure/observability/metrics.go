package observability

import (
	"log/slog"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics for auth operations.
var (
	RegistrationTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auth_registration_total",
		Help: "Total number of user registrations",
	})
	RegistrationErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auth_registration_errors_total",
		Help: "Total number of failed registrations",
	})
	LoginTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auth_login_total",
		Help: "Total number of login attempts",
	})
	LoginErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auth_login_errors_total",
		Help: "Total number of failed logins",
	})
	PasswordResetRequestTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auth_password_reset_request_total",
		Help: "Total number of password reset requests",
	})
	TokenRefreshTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auth_token_refresh_total",
		Help: "Total number of token refresh requests",
	})
	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "auth_request_duration_seconds",
		Help:    "Duration of auth requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"method"})
	RateLimitHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auth_rate_limit_hits_total",
		Help: "Total number of rate limit hits",
	}, []string{"action"})
)

// NewLogger creates a structured JSON logger.
func NewLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     lvl,
		AddSource: true,
	})
	return slog.New(handler)
}
