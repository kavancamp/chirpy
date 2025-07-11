package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"database/sql"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	DB             *database.Queries
	Platform        string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) adminMetricsHandler(w http.ResponseWriter, r *http.Request) {
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

func (cfg *apiConfig) adminResetHandler(w http.ResponseWriter, r *http.Request) {
	if cfg.Platform != "dev" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	err := cfg.DB.DeleteAllUsers(r.Context())
	if err != nil {
		log.Printf("Error deleting users: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to reset users")
		return
	}
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, "Hits counter reset to 0\n")
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	resp := map[string]string{"error": msg}
	respondWithJSON(w, code, resp)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	d, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling response: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Something went wrong"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(d)
}
func cleanProfanity(input string) string {
	profaneWords := []string{"kerfuffle", "sharbert", "fornax"}
	words := strings.Split(input, " ")
	for i, word := range words {
		lower := strings.ToLower(word)
		for _, profane := range profaneWords {
			if lower == profane {
				words[i] = "****"
			}
		}
	}
	return strings.Join(words, " ")
}
func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	type ChirpInput struct {
		Body string `json:"body"`
	}
	type CleanedResponse struct {
		CleanedBody string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(r.Body)
	var input ChirpInput
	if err := decoder.Decode(&input); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}
	if len(input.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}
	cleaned := cleanProfanity(input.Body)
	resp := CleanedResponse{CleanedBody: cleaned}
	respondWithJSON(w, http.StatusOK, resp)
}

func (cfg *apiConfig) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	type userInput struct {
		Email string `json:"email"`
	}

	var input userInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Request")
		return
	}
	if strings.TrimSpace(input.Email) == "" {
		respondWithError(w, http.StatusBadRequest, "Email is required")
		return
	}

	dbUser, err := cfg.DB.CreateUser(r.Context(), input.Email)
	if err != nil {
		log.Printf("Error creating user: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not create user")
		return
	}
	user := User {
		ID: dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email: dbUser.Email,
	}
	respondWithJSON(w, http.StatusCreated, user)
}



func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Connect to DB
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	dbQueries := database.New(db)
	cfg := apiConfig{
		DB: dbQueries,
		Platform: os.Getenv("PLATFORM"),
	}
	mux := http.NewServeMux()
	//readiness endpoint
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	mux.HandleFunc("GET /admin/metrics", cfg.adminMetricsHandler)
	mux.HandleFunc("POST /admin/reset", cfg.adminResetHandler)
	mux.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)
	mux.HandleFunc("POST /api/users", cfg.handleCreateUser)
	mux.HandleFunc("POST /api/chirps", cfg.handleChirps)
	// File server wrapped with metrics middleware
	fileServer := http.FileServer(http.Dir("."))
	mux.Handle("/app/", http.StripPrefix("/app", cfg.middlewareMetricsInc(fileServer)))

	// start the server on port 8089
	log.Println("Server running on http://localhost:8080")
	err = http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}