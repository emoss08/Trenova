package middleware

import (
	"context"
	"net/http"

	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/session"
	"github.com/emoss08/trenova/tools/types"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "userID"
	ContextKeyOrgID  contextKey = "orgID"
	ContextKeyBuID   contextKey = "buID"
)

func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		store, storeErr := session.GetStore()
		if storeErr != nil {
			return
		}

		userID, orgID, buID, ok := tools.GetSessionDetails(r, store)
		if !ok {
			tools.ResponseWithError(w, http.StatusUnauthorized, types.ValidationErrorDetail{
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
