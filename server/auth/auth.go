package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/romain325/doc-thor/server/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type contextKey string

const ctxUser contextKey = "user"

var ErrUnauthorized = errors.New("unauthorized")

// HashPassword returns a bcrypt hash of the given password.
func HashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// CheckPassword compares a plaintext password against a bcrypt hash.
func CheckPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// GenerateToken returns the raw (plaintext) token and its SHA-256 hex hash.
// Callers store only the hash; the raw value is sent to the client once.
func GenerateToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	raw = hex.EncodeToString(b)
	hash = HashToken(raw)
	return
}

// HashToken returns the SHA-256 hex digest of a raw token string.
func HashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// UserFromContext returns the authenticated user injected by RequireAuth, or nil.
func UserFromContext(ctx context.Context) *models.User {
	u, _ := ctx.Value(ctxUser).(*models.User)
	return u
}

// RequireAuth is a chi-compatible middleware. It extracts a Bearer token,
// validates it against the DB, and injects the owning User into the context.
func RequireAuth(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := extractBearer(r)
			if raw == "" {
				denyJSON(w)
				return
			}
			user, err := validateToken(db, raw)
			if err != nil {
				denyJSON(w)
				return
			}
			ctx := context.WithValue(r.Context(), ctxUser, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(h, "Bearer ")
}

func validateToken(db *gorm.DB, raw string) (*models.User, error) {
	hash := HashToken(raw)
	var token models.Token
	if err := db.Where("token_hash = ?", hash).First(&token).Error; err != nil {
		return nil, ErrUnauthorized
	}
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		db.Delete(&token)
		return nil, ErrUnauthorized
	}
	var user models.User
	if err := db.First(&user, token.UserID).Error; err != nil {
		return nil, ErrUnauthorized
	}
	return &user, nil
}

func denyJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"unauthorized"}`)) //nolint:errcheck
}
