package api

import (
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Email == "" || req.Password == "" {
			respondError(w, http.StatusBadRequest, "Email and password are required")
			return
		}

		// Hash password
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to process password")
			return
		}

		// Create user (defaults to 'free' plan)
		id, err := s.userStore.CreateUser(r.Context(), req.Email, string(hash), "free")
		if err != nil {
			s.logger.Error("failed to create user", "error", err)
			respondError(w, http.StatusConflict, "User already exists or database error")
			return
		}

		// Automatically log them in by issuing a token
		token, err := s.generateToken(id)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		respondJSON(w, http.StatusCreated, map[string]interface{}{
			"message": "Registration successful",
			"user_id": id,
			"token":   token,
		})
	}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		u, err := s.userStore.GetUserByEmail(r.Context(), req.Email)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Database error")
			return
		}
		if u == nil {
			respondError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
			respondError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}

		token, err := s.generateToken(u.ID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Login successful",
			"user_id": u.ID,
			"plan":    u.Plan,
			"token":   token,
		})
	}
}

func (s *Server) handleGetMe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		// For simplicity, we just return the ID. A real implementation would fetch from DB.
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"user_id": userID,
		})
	}
}
