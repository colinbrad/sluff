package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/colinbradley/sluff/internal/model"
	"github.com/colinbradley/sluff/internal/store"
)

type AuthHandler struct {
	store     store.Store
	jwtSecret string
}

func NewAuthHandler(s store.Store, jwtSecret string) *AuthHandler {
	return &AuthHandler{store: s, jwtSecret: jwtSecret}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}
	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	existing, err := h.store.GetGuideByUsername(req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check username")
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, "username already taken")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	guide := &model.Guide{
		ID:           uuid.New().String(),
		Username:     req.Username,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}
	if err := h.store.CreateGuide(guide); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	token, err := h.generateToken(guide.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"token": token,
		"guide": guide,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	guide, err := h.store.GetGuideByUsername(req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to look up account")
		return
	}
	if guide == nil {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(guide.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := h.generateToken(guide.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"guide": guide,
	})
}

func (h *AuthHandler) generateToken(guideID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": guideID,
		"exp": time.Now().Add(30 * 24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}
