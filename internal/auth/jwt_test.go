package auth

import (
	"testing"
	"time"
	"net/http"
	"github.com/google/uuid"
)

const (
	testSecret     = "supersecretkey"
	wrongSecret    = "wrongsecretkey"
	validDuration  = time.Minute
	expiredDuration = -time.Minute
)

func TestMakeAndValidateJWT_Success(t *testing.T) {
	userID := uuid.New()

	token, err := MakeJWT(userID, testSecret, validDuration)
	if err != nil {
		t.Fatalf("expected no error making token, got: %v", err)
	}

	returnedID, err := ValidateJWT(token, testSecret)
	if err != nil {
		t.Fatalf("expected no error validating token, got: %v", err)
	}

	if returnedID != userID {
		t.Errorf("expected userID %v, got %v", userID, returnedID)
	}
}

func TestValidateJWT_WrongSecret(t *testing.T) {
	userID := uuid.New()

	token, err := MakeJWT(userID, testSecret, validDuration)
	if err != nil {
		t.Fatalf("error creating token: %v", err)
	}

	_, err = ValidateJWT(token, wrongSecret)
	if err == nil {
		t.Fatal("expected error when using wrong secret, got nil")
	}
}

func TestValidateJWT_ExpiredToken(t *testing.T) {
	userID := uuid.New()

	token, err := MakeJWT(userID, testSecret, expiredDuration)
	if err != nil {
		t.Fatalf("error creating expired token: %v", err)
	}

	_, err = ValidateJWT(token, testSecret)
	if err == nil {
		t.Fatal("expected error from expired token, got nil")
	}
}

func TestGetBearerToken(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer test-token")

	token, err := GetBearerToken(headers)
	if err != nil || token != "test-token" {
		t.Fatalf("expected 'test-token', got '%s', err: %v", token, err)
	}
}