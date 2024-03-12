package utils

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"trenova/app/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/wader/gormstore/v2"
)

type ValidationError struct {
	Response models.ValidationErrorResponse
}

func (ve ValidationError) Error() string {
	errBytes, _ := json.Marshal(ve.Response)
	return string(errBytes)
}

func ParseBody(r *http.Request, body any) error {
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(body); err != nil {
		return err
	}
	return nil
}

// ParseBodyAndValidate parses the request body into the given struct and validates it using the given validator.
// If the body is invalid, it writes a 400 response with the validation error.
func ParseBodyAndValidate(validator *Validator, w http.ResponseWriter, r *http.Request, body any) error {
	if err := ParseBody(r, body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	if err := validator.Validate(body); err != nil {
		var validationErr *ValidationError
		if errors.As(err, &validationErr) {
			errorBytes, _ := json.Marshal(validationErr.Response)
			http.Error(w, string(errorBytes), http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return err
	}

	return nil
}

// GetMuxVar retrieves a variable from the route pattern match and writes an error if it's not found.
func GetMuxVar(w http.ResponseWriter, r *http.Request, key string) string {
	vars := mux.Vars(r)
	value, ok := vars[key]
	if !ok {
		ResponseWithError(w, http.StatusBadRequest, models.ValidationErrorDetail{
			Code:   "invalid",
			Detail: "The required parameter is missing.",
			Attr:   key,
		})
		value = ""
	}
	return value
}

// RegisterGob registers the UUID type with gob, so it can be used in sessions.
func RegisterGob() {
	gob.Register(uuid.UUID{})
	gob.Register(models.User{})
}

func GetSystemSessionID() string {
	key := os.Getenv("SESSION_ID")
	if key == "" {
		log.Fatal("SESSION_ID not found in environment")
	}

	return key
}

// GetSessionDetails retrieves user ID, organization ID, and business unit ID from the session.
func GetSessionDetails(r *http.Request, store *gormstore.Store) (uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	if store == nil {
		log.Println("Session store is not initialized")
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	sessionID := GetSystemSessionID()
	session, err := store.Get(r, sessionID)
	if err != nil {
		log.Printf("Error retrieving session: %v", err)
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	userID, userOk := session.Values["userID"].(uuid.UUID)
	orgID, orgOk := session.Values["organizationID"].(uuid.UUID)
	buID, buOk := session.Values["businessUnitID"].(uuid.UUID)

	return userID, orgID, buID, userOk && orgOk && buOk
}
