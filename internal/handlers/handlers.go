package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/kavancamp/chirpy/internal/auth"
	"github.com/kavancamp/chirpy/internal/database"
)

type ApiConfig struct {
	fileserverHits atomic.Int32
	DB             *database.Queries
	Platform        string
	JWTSecret string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func (cfg *ApiConfig) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *ApiConfig) AdminMetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	count := cfg.fileserverHits.Load()
	html := fmt.Sprintf(`
	<html>
		<body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		</body>
	</html>
	`, count)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, html)
}

func (cfg *ApiConfig) AdminResetHandler(w http.ResponseWriter, r *http.Request) {
	if cfg.Platform != "dev" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	err := cfg.DB.DeleteAllUsers(r.Context())
	if err != nil {
		log.Printf("Error deleting users: %s", err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to reset users")
		return
	}
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, "Hits counter reset to 0\n")
}

func (cfg *ApiConfig) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	type userInput struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}
	
	var input userInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid Request")
		return
	}
	if strings.TrimSpace(input.Email) == "" {
		RespondWithError(w, http.StatusBadRequest, "Email is required")
		return
	}
	if strings.TrimSpace(input.Password) == "" {
		RespondWithError(w, http.StatusBadRequest, "Password is required")
		return
	}
	hashed, err := auth.HashPassword(input.Password)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}
	dbUser, err := cfg.DB.CreateUser(r.Context(), database.CreateUserParams{
		Email: input.Email,
		HashedPassword: hashed,
	})
	if err != nil {
		log.Printf("Error creating user: %s", err)
		RespondWithError(w, http.StatusInternalServerError, "Could not create user")
		return
	}
	user := User {
		ID: dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email: dbUser.Email,
	}
	RespondWithJSON(w, http.StatusCreated, user)
}

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
	type Chirp struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body     string    `json:"body"`
		UserID uuid.UUID	`json:"user_id"`
	}
	var chirpList []Chirp

	for _, c := range chirps {
		chirpList = append(chirpList, Chirp{
		ID: c.ID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		Body: c.Body,
		UserID: c.UserID,

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

func (cfg *ApiConfig) HandleLogin(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Email            string `json:"email"`
		Password         string `json:"password"`
		ExpiresInSeconds int    `json:"expires_in_seconds"`
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

	expires := time.Hour
	if body.ExpiresInSeconds > 0 && body.ExpiresInSeconds <= 3600 {
		expires = time.Duration(body.ExpiresInSeconds) * time.Second
	}

	token, err := auth.MakeJWT(dbUser.ID, cfg.JWTSecret, expires)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to create token")
		return
	}
	RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":         dbUser.ID,
		"email":      dbUser.Email,
		"created_at": dbUser.CreatedAt,
		"updated_at": dbUser.UpdatedAt,
		"token":      token,
	})
}