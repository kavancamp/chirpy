package main

import (
	"github.com/kavancamp/chirpy/internal/database"
	"github.com/kavancamp/chirpy/internal/handlers"
	"database/sql"
	"net/http"
	"fmt"

	"os"
	"log"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)



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
	jwtSecret := os.Getenv("JWT_SECRET")
	cfg := handlers.ApiConfig{
		DB: dbQueries,
		Platform: os.Getenv("PLATFORM"),
		JWTSecret: jwtSecret,
	}
	mux := http.NewServeMux()
	//readiness endpoint
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	mux.HandleFunc("GET /admin/metrics", cfg.AdminMetricsHandler)
	mux.HandleFunc("POST /admin/reset", cfg.AdminResetHandler)
	mux.HandleFunc("POST /api/users", cfg.HandleCreateUser)

	mux.HandleFunc("POST /api/chirps", cfg.HandleCreateChirp)
	mux.HandleFunc("GET /api/chirps", cfg.HandleGetChirps)
	mux.HandleFunc("GET /api/chirps/", cfg.HandleGetChirpByID)
	mux.HandleFunc("POST /api/login", cfg.HandleLogin)

	// File server wrapped with metrics middleware
	fileServer := http.FileServer(http.Dir("."))
	mux.Handle("/app/", http.StripPrefix("/app", cfg.MiddlewareMetricsInc(fileServer)))

	// start the server on port 8089
	log.Println("Server running on http://localhost:8080")
	err = http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}