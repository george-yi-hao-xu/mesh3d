package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	authCookieName = "mesh3d_token"
	tokenTTL       = 24 * time.Hour
)

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"createdAt"`
}

type jwtClaims struct {
	Subject   string `json:"sub"`
	Username  string `json:"username"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

type contextKey string

const userContextKey contextKey = "user"

var errInvalidToken = errors.New("invalid token")

func initJWTSecret() []byte {
	secret := strings.TrimSpace(os.Getenv("MESH3D_JWT_SECRET"))
	if secret != "" {
		return []byte(secret)
	}

	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		log.Printf("warning: could not generate random JWT secret, falling back to timestamp secret: %v", err)
		return []byte(time.Now().UTC().Format(time.RFC3339Nano))
	}
	log.Printf("warning: MESH3D_JWT_SECRET is not set; using a process-local development secret")
	return buf[:]
}

func userToResponse(user *User) userResponse {
	if user == nil {
		return userResponse{}
	}
	return userResponse{
		ID:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	}
}

func createJWT(secret []byte, user *User, now time.Time) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	claims := jwtClaims{
		Subject:   user.ID,
		Username:  user.Username,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(tokenTTL).Unix(),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	enc := base64.RawURLEncoding
	unsigned := enc.EncodeToString(headerJSON) + "." + enc.EncodeToString(claimsJSON)
	signature := signJWT(secret, unsigned)
	return unsigned + "." + enc.EncodeToString(signature), nil
}

func verifyJWT(secret []byte, token string, now time.Time) (jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return jwtClaims{}, errInvalidToken
	}

	unsigned := parts[0] + "." + parts[1]
	expectedSig := signJWT(secret, unsigned)
	actualSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(actualSig, expectedSig) {
		return jwtClaims{}, errInvalidToken
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return jwtClaims{}, errInvalidToken
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil || header.Alg != "HS256" || header.Typ != "JWT" {
		return jwtClaims{}, errInvalidToken
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return jwtClaims{}, errInvalidToken
	}
	var claims jwtClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return jwtClaims{}, errInvalidToken
	}
	if claims.Subject == "" || claims.Username == "" || claims.ExpiresAt <= now.Unix() {
		return jwtClaims{}, errInvalidToken
	}
	return claims, nil
}

func signJWT(secret []byte, unsigned string) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(unsigned))
	return mac.Sum(nil)
}

func setAuthCookie(w http.ResponseWriter, r *http.Request, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isSecureRequest(r),
	})
}

func clearAuthCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isSecureRequest(r),
	})
}

func isSecureRequest(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
