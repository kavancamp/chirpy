package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
	
	"github.com/google/uuid"
	"github.com/kavancamp/chirpy/internal/auth"
	"github.com/kavancamp/chirpy/internal/database"
)

type InsertRefreshTokenParams struct {
    Token     string
    UserID    uuid.UUID
    ExpiresAt time.Time
}

func (cfg *ApiConfig) HandleLogin(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var body requestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	dbUser, err := cfg.DB.GetUserByEmail(r.Context(), body.Email)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}

	if err := auth.CheckPasswordHash(body.Password, dbUser.HashedPassword); err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}

	accessToken, err := auth.MakeJWT(dbUser.ID, cfg.JWTSecret, time.Hour)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to create access token")
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to create refresh token")
		return
	}

	expiresAt := time.Now().Add(60 * 24 * time.Hour) // 60 days
	err = cfg.DB.InsertRefreshToken(r.Context(), database.InsertRefreshTokenParams{
		Token:     refreshToken,
		UserID:    dbUser.ID,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to store refresh token")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":            dbUser.ID,
		"email":         dbUser.Email,
		"created_at":    dbUser.CreatedAt,
		"updated_at":    dbUser.UpdatedAt,
		"token":         accessToken,
		"refresh_token": refreshToken,
	})
}

func (cfg *ApiConfig) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	tokenStr, err := auth.GetBearerToken(r.Header)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Missing or invalid token")
		return
	}

	refreshToken, err := cfg.DB.GetUserFromRefreshToken(r.Context(), tokenStr)
	if err != nil || isRevokedOrExpired(refreshToken) {
		RespondWithError(w, http.StatusUnauthorized, "Refresh token is invalid or expired")
		return
	}

	accessToken, err := auth.MakeJWT(refreshToken.UserID, cfg.JWTSecret, time.Hour)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Could not generate token")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{
		"token": accessToken,
	})
}

func (cfg *ApiConfig) HandleRevoke(w http.ResponseWriter, r *http.Request) {
	tokenStr, err := auth.GetBearerToken(r.Header)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Missing or invalid token")
		return
	}

	now := time.Now().UTC()
	err = cfg.DB.RevokeRefreshToken(r.Context(), database.RevokeRefreshTokenParams{
		RevokedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt: now,
		Token:     tokenStr,
	})
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Token not found or could not be revoked")
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content
}

func isRevokedOrExpired(token database.GetUserFromRefreshTokenRow) bool {
	return token.RevokedAt.Valid || time.Now().After(token.ExpiresAt)
}