package handlers
import (
	"encoding/json"
	"net/http"
	"log"
	"strings"
)

func RespondWithError(w http.ResponseWriter, code int, msg string) {
	resp := map[string]string{"error": msg}
	RespondWithJSON(w, code, resp)
}

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
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
func CleanProfanity(input string) string {
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

