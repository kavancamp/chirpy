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