package tools

import (
	"encoding/gob"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/emoss08/trenova/tools/session"
	"github.com/goccy/go-json"

	"github.com/google/uuid"
)

func ParseBody(r *http.Request, body any) error {
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(body); err != nil {
		return err
	}
	return nil
}

var validatorInstance *Validator

func init() {
	var err error
	validatorInstance, err = NewValidator()
	if err != nil {
		log.Fatalf("Failed to initialize validator: %v", err)
	}
}

// ParseBodyAndValidate parses the request body into the given struct and validates it using the given validator.
// If the body is invalid, it writes a 400 response with the validation error.
func ParseBodyAndValidate(w http.ResponseWriter, r *http.Request, body any) error {
	if err := ParseBody(r, body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	if err := validatorInstance.Validate(body); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		var validationErr *ValidationError
		if errors.As(err, &validationErr) {
			json.NewEncoder(w).Encode(validationErr.Response)
		} else {
			// Generic error response
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		}
		return err
	}

	return nil
}

// RegisterGob registers the UUID type with gob, so it can be used in sessions.
func RegisterGob() {
	gob.Register(uuid.UUID{})
}

func GetSystemSessionName() string {
	key := os.Getenv("SESSION_NAME")
	if key == "" {
		log.Fatal("SESSION_NAME not found in environment")
	}

	return key
}

// GetSessionDetails retrieves user ID, organization ID, and business unit ID from the session.
func GetSessionDetails(r *http.Request, store *session.Store) (uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	if store == nil {
		log.Println("Session store is not initialized")
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	sessionName := GetSystemSessionName()
	session, err := store.Get(r, sessionName)
	if err != nil {
		log.Printf("Error retrieving session: %v", err)
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	userID, userOk := session.Values["userID"].(uuid.UUID)
	orgID, orgOk := session.Values["organizationID"].(uuid.UUID)
	buID, buOk := session.Values["businessUnitID"].(uuid.UUID)

	return userID, orgID, buID, userOk && orgOk && buOk
}
