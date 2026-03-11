package graphql

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	gql "github.com/graphql-go/graphql"
)

type Handler struct {
	schema gql.Schema
}

func NewHandler(schema gql.Schema) *Handler {
	return &Handler{schema: schema}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var params struct {
		Query         string                 `json:"query"`
		OperationName string                 `json:"operationName"`
		Variables     map[string]interface{} `json:"variables"`
	}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Extract auth token from Authorization header
	ctx := r.Context()
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		ctx = context.WithValue(ctx, contextKeyToken, token)
	}

	// Pass IP and User-Agent for session/rate-limiting
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}
	ctx = context.WithValue(ctx, contextKeyIP, ip)
	ctx = context.WithValue(ctx, contextKeyUserAgent, r.UserAgent())

	result := gql.Do(gql.Params{
		Schema:         h.schema,
		RequestString:  params.Query,
		OperationName:  params.OperationName,
		VariableValues: params.Variables,
		Context:        ctx,
	})

	w.Header().Set("Content-Type", "application/json")
	if len(result.Errors) > 0 {
		w.WriteHeader(http.StatusOK) // GraphQL returns 200 even for errors per spec
	}
	json.NewEncoder(w).Encode(result)
}
