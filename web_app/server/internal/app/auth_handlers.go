package app

import (
	"encoding/json"
	"net/http"
	"time"
)

func (a *App) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	user, err := a.store.CreateUser(req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	now := time.Now().UTC()
	token, err := createJWT(a.jwtSecret, user, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create token")
		return
	}
	setAuthCookie(w, r, token, now.Add(tokenTTL))
	writeJSON(w, http.StatusCreated, map[string]userResponse{"user": userToResponse(user)})
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	user, err := a.store.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	now := time.Now().UTC()
	token, err := createJWT(a.jwtSecret, user, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create token")
		return
	}
	setAuthCookie(w, r, token, now.Add(tokenTTL))
	writeJSON(w, http.StatusOK, map[string]userResponse{"user": userToResponse(user)})
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clearAuthCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user, ok := a.authenticateRequest(r)
	if !ok {
		clearAuthCookie(w, r)
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]userResponse{"user": userToResponse(user)})
}
