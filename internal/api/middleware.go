package api

import (
	"context"
	"net/http"

	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
)

type ctxKey string

const maxUserCtxKey ctxKey = "max_user"

func ValidateMaxInitData(botToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			initData := r.Header.Get("X-Max-Init-Data")
			if initData == "" {
				http.Error(w, `{"error":"missing init data"}`, http.StatusUnauthorized)
				return
			}

			data, err := maxbot.ValidateInitData(initData, botToken)
			if err != nil {
				http.Error(w, `{"error":"invalid init data"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), maxUserCtxKey, data)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
