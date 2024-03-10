package middleware

import (
	"context"
	"net/http"
	"trenova-go-backend/utils"

	"github.com/wader/gormstore/v2"
)

func SessionMiddleware(store *gormstore.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, orgID, buID, ok := utils.GetSessionDetails(r, store)
			if !ok {
				utils.ResponseWithError(w, http.StatusUnauthorized, utils.ValidationErrorDetail{
					Code:   "unauthorized",
					Detail: "Unauthorized",
					Attr:   "session",
				})
				return
			}

			// Store the session details in the request context for later retrieval
			ctx := context.WithValue(r.Context(), "userID", userID)
			ctx = context.WithValue(ctx, "orgID", orgID)
			ctx = context.WithValue(ctx, "buID", buID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
