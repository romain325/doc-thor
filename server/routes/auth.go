package routes

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/romain325/doc-thor/server/auth"
	"github.com/romain325/doc-thor/server/models"
	"gorm.io/gorm"
)

func Login(db *gorm.DB, sessionTTLHours int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		var user models.User
		if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		if err := auth.CheckPassword(req.Password, user.PasswordHash); err != nil {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}

		raw, hash, err := auth.GenerateToken()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "token generation failed")
			return
		}

		exp := time.Now().Add(time.Duration(sessionTTLHours) * time.Hour)
		token := models.Token{
			UserID:    user.ID,
			TokenHash: hash,
			Type:      "session",
			ExpiresAt: &exp,
		}
		if err := db.Create(&token).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"token": raw})
	}
}

func CreateAPIKey(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var req struct {
			Label string `json:"label"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		raw, hash, err := auth.GenerateToken()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "token generation failed")
			return
		}

		token := models.Token{
			UserID:    user.ID,
			TokenHash: hash,
			Type:      "apikey",
			Label:     req.Label,
		}
		if err := db.Create(&token).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{"key": raw, "label": req.Label})
	}
}

func GetMe(_ *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		writeJSON(w, http.StatusOK, user)
	}
}
