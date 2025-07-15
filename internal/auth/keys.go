package auth

import (
	"errors"
	"net/http"
	"strings"
)

func GetAPIKey(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if !strings.HasPrefix(authHeader, "ApiKey ") {
		return "", errors.New("missing or invalid Authorization header")
	}
	return strings.TrimPrefix(authHeader, "ApiKey "), nil
}