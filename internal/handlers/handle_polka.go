package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/kavancamp/chirpy/internal/auth"
)
func (cfg *ApiConfig) HandlePolkaWebhook(w http.ResponseWriter, r *http.Request) {
	apiKey, err := auth.GetAPIKey(r.Header) 
	if err != nil || apiKey != cfg.PolkaKey {
		RespondWithError(w, http.StatusUnauthorized, "Invalid API Key")
	} 

	type webhookRequest struct {
		Event string `json:"event"`
		Data struct {
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}

	var req webhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if req.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent) // We don't care about other events
		return
	}

	err = cfg.DB.UpgradeUserToChirpyRed(r.Context(), req.Data.UserID)
	if err != nil {
		RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content
}
