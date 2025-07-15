package handlers

import (
	"encoding/json"
	"net/http"
	"time"
	"log"
	"strings"
	"sort"
	"github.com/kavancamp/chirpy/internal/auth"
	"github.com/kavancamp/chirpy/internal/database"
	"github.com/google/uuid"
)
func (cfg *ApiConfig) HandleCreateChirp(w http.ResponseWriter, r *http.Request) {
	type ChirpInput struct {
		Body string `json:"body"`
	}

	var input ChirpInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if len(input.Body) > 140 {
		RespondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	tokenStr, err := auth.GetBearerToken(r.Header)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Missing or invalid token")
		return
	}

	userID, err := auth.ValidateJWT(tokenStr, cfg.JWTSecret)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	body := CleanProfanity(input.Body)
	now := time.Now().UTC()
	id := uuid.New()

	dbChirp, err := cfg.DB.CreateChirp(r.Context(), database.CreateChirpParams{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		Body:      body,
		UserID:    userID,
	})
	if err != nil {
		log.Printf("error creating chirp: %s", err)
		RespondWithError(w, http.StatusInternalServerError, "Could not create chirp")
		return
	}
	type ChirpResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}
	RespondWithJSON(w, http.StatusCreated, ChirpResponse{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	})
	}

func (cfg *ApiConfig) HandleGetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.DB.GetChirps(r.Context())
	if err != nil {
		log.Printf("error getting chirps: %s", err)
		RespondWithError(w, http.StatusInternalServerError, "Could not retrieve chirps")
		return
	}

	authorIDStr := r.URL.Query().Get("author_id")
	if authorIDStr != "" {
		authorID, err := uuid.Parse(authorIDStr)
		if err == nil {
			filtered := make([]database.Chirp, 0)
			for _, c := range chirps {
				if c.UserID == authorID {
					filtered = append(filtered, c)
				}
			}
			chirps = filtered
		}
	}

	sortOrder := r.URL.Query().Get("sort")
	if sortOrder != "desc" {
		sort.Slice(chirps, func(i, j int) bool {
			return chirps[i].CreatedAt.Before(chirps[j].CreatedAt)
		})
	} else {
		sort.Slice(chirps, func(i, j int) bool {
			return chirps[j].CreatedAt.Before(chirps[i].CreatedAt)
		})
	}

	type Chirp struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}

	var chirpList []Chirp
	for _, c := range chirps {
		chirpList = append(chirpList, Chirp{
			ID:        c.ID,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
			Body:      c.Body,
			UserID:    c.UserID,
		})
	}

	RespondWithJSON(w, http.StatusOK, chirpList)
}


func (cfg *ApiConfig) HandleGetChirpByID(w http.ResponseWriter, r *http.Request) {
	//get id from url
	idStr := strings.TrimPrefix(r.URL.Path, "/api/chirps/")
	chirpID, err := uuid.Parse(idStr)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "invalid chirp ID")
		return
	}
	//get chirp from database
	dbChirp, err := cfg.DB.GetChirpsByID(r.Context(), chirpID)
	if err != nil {
		RespondWithError(w, http.StatusNotFound, "chirp not found")
		return
	}
	// Map to output struct with correct JSON field names
	type Chirp struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}
	chirp := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}

	RespondWithJSON(w, http.StatusOK, chirp)
}
func (cfg *ApiConfig) HandleDeleteChirp(w http.ResponseWriter, r *http.Request) {
	// 1. Extract and validate JWT
	tokenStr, err := auth.GetBearerToken(r.Header)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Missing or invalid token")
		return
	}

	userID, err := auth.ValidateJWT(tokenStr, cfg.JWTSecret)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	// 2. Get chirp ID from URL
	idStr := strings.TrimPrefix(r.URL.Path, "/api/chirps/")
	chirpID, err := uuid.Parse(idStr)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}

	// 3. Get chirp from DB
	chirp, err := cfg.DB.GetChirpsByID(r.Context(), chirpID)
	if err != nil {
		RespondWithError(w, http.StatusNotFound, "Chirp not found")
		return
	}

	// 4. Ensure the authenticated user is the author
	if chirp.UserID != userID {
		RespondWithError(w, http.StatusForbidden, "Not authorized to delete this chirp")
		return
	}

	// 5. Delete the chirp
	err = cfg.DB.DeleteChirpByID(r.Context(), chirpID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to delete chirp")
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content
}
