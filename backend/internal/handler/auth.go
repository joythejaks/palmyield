package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/joythejaks/palmyield/backend/internal/middleware"
	"github.com/joythejaks/palmyield/backend/internal/service"
)

type AuthHandler struct {
	Service *service.AuthService
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type tokenPairResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Identifier == "" || req.Password == "" {
		http.Error(w, "identifier and password are required", http.StatusBadRequest)
		return
	}

	tokens, err := h.Service.Login(r.Context(), req.Identifier, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
		case errors.Is(err, service.ErrAccountDisabled):
			http.Error(w, "account disabled", http.StatusForbidden)
		default:
			slog.Error("login failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, http.StatusOK, tokenPairResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		http.Error(w, "refresh_token is required", http.StatusBadRequest)
		return
	}

	tokens, err := h.Service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRefreshToken) {
			http.Error(w, "invalid or expired refresh token", http.StatusUnauthorized)
			return
		}
		slog.Error("refresh failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, tokenPairResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		http.Error(w, "refresh_token is required", http.StatusBadRequest)
		return
	}

	if err := h.Service.Logout(r.Context(), req.RefreshToken); err != nil {
		slog.Error("logout failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type inviteRequest struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
	Role  string `json:"role"`
}

type inviteResponse struct {
	UserID       string `json:"user_id"`
	TempPassword string `json:"temp_password"`
}

// Invite is admin-only (enforced by RequireRole middleware). The new user is
// always created in the inviting admin's own cooperative.
func (h *AuthHandler) Invite(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req inviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Role != "admin" && req.Role != "farmer" {
		http.Error(w, "role must be admin or farmer", http.StatusBadRequest)
		return
	}
	if req.Email == "" && req.Phone == "" {
		http.Error(w, "email or phone is required", http.StatusBadRequest)
		return
	}

	result, err := h.Service.Invite(r.Context(), claims.CooperativeID, req.Email, req.Phone, req.Role)
	if err != nil {
		slog.Error("invite failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, inviteResponse{UserID: result.UserID, TempPassword: result.TempPassword})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to write JSON response", "error", err)
	}
}
