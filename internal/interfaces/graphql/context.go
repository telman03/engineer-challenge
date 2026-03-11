package graphql

type contextKey string

const (
	contextKeyToken     contextKey = "token"
	contextKeyIP        contextKey = "ip"
	contextKeyUserAgent contextKey = "user_agent"
)
