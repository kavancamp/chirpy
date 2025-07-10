package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32

}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Add(1)
	next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) adminMetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
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
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, "Hits counter reset to 0\n")
}

func writeJSON(w http.ResponseWriter, data interface{}){
	w.Header().Set("Content-Type", "application/json")
	d, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Something went wrong"}`))
		return
	}
	w.Write(d)
}

func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": "Something went wrong"})
		return
	}
	if len(params.Body) > 140 {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "Chirp is too long"})
		return
	}
	w.WriteHeader(http.StatusOK)
	writeJSON(w, map[string]bool{"valid": true})
}
func main(){
	
 	// // Create a new ServeMux (router)
	mux := http.NewServeMux()
	cfg := apiConfig{}

	//readiness endpoint 
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request){
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})
	mux.HandleFunc("GET /admin/metrics", cfg. adminMetricsHandler)
	mux.HandleFunc("POST /admin/reset", cfg.adminResetHandler)
	mux.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)
	// File server wrapped with metrics middleware
	fileServer := http.FileServer(http.Dir("."))
	mux.Handle("/app/", http.StripPrefix("/app", cfg.middlewareMetricsInc(fileServer)))

	// start the server on port 8089
	http.ListenAndServe(":8080", mux)

}

