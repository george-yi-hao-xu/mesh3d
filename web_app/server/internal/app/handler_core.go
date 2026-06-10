package app

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// handleHealth reports whether the server process is reachable.
func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := a.authenticateRequest(r)
		if !ok {
			clearAuthCookie(w, r)
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next(w, r.WithContext(ctx))
	}
}

func (a *App) authenticateRequest(r *http.Request) (*User, bool) {
	cookie, err := r.Cookie(authCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return nil, false
	}

	claims, err := verifyJWT(a.jwtSecret, cookie.Value, time.Now().UTC())
	if err != nil {
		return nil, false
	}
	user, ok := a.store.GetUser(claims.Subject)
	if !ok || user.Username != claims.Username {
		return nil, false
	}
	return user, true
}

func currentUser(r *http.Request) *User {
	user, _ := r.Context().Value(userContextKey).(*User)
	return user
}
