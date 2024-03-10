package middleware

import (
	"context"
	"net/http"
	"trenova/app/models"
	"trenova/utils"

	"github.com/wader/gormstore/v2"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "userID"
	ContextKeyOrgID  contextKey = "orgID"
	ContextKeyBuID   contextKey = "buID"
)

func SessionMiddleware(store *gormstore.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, orgID, buID, ok := utils.GetSessionDetails(r, store)
			if !ok {
				utils.ResponseWithError(w, http.StatusUnauthorized, models.ValidationErrorDetail{
					Code:   "unauthorized",
					Detail: "Unauthorized",
					Attr:   "session",
				})
				return
			}

			// Store the session details in the request context for later retrieval
			ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
			ctx = context.WithValue(ctx, ContextKeyOrgID, orgID)
			ctx = context.WithValue(ctx, ContextKeyBuID, buID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
