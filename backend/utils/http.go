package utils

import (
	"encoding/gob"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"trenova-go-backend/app/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/wader/gormstore/v2"
)

func ParseBody(r *http.Request, body interface{}) error {
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(body); err != nil {
		return err
	}
	return nil
}

func ParseBodyAndValidate(w http.ResponseWriter, r *http.Request, body interface{}) error {
	if err := ParseBody(r, body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	if err := Validate(body); err != nil {
		var validationErr ValidationErrorResponse
		if jsonErr := json.Unmarshal([]byte(err.Error()), &validationErr); jsonErr == nil {
			errorBytes, _ := json.Marshal(validationErr)
			http.Error(w, string(errorBytes), http.StatusBadRequest)
		} else {
			// Fallback in case the error is not a validation error
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return err
	}

	return nil
}

// GetUser is a helper function for getting the authenticated user's ID from the context.
func GetUser(r *http.Request) *uint {
	if id, ok := r.Context().Value("USER").(uint); ok {
		return &id
	}
	return nil
}

// UserFromContext is a helper function to extract the user model from the request context.
func UserFromContext(r *http.Request) *models.User {
	if user, ok := r.Context().Value("user").(*models.User); ok {
		return user
	}
	return nil
}

// GetMuxVar retrieves a variable from the route pattern match and writes an error if it's not found.
func GetMuxVar(w http.ResponseWriter, r *http.Request, key string) (value string) {
	vars := mux.Vars(r)
	value, ok := vars[key]
	if !ok {
		ResponseWithError(w, http.StatusBadRequest, ValidationErrorDetail{
			Code:   "invalid",
			Detail: "The required parameter is missing.",
			Attr:   key,
		})
		value = "" // Return an empty string if the value is not found
	}
	return value
}

// RegisterGob registers the UUID type with gob so it can be used in sessions.
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
func GetUserIDFromSession(r *http.Request, store *gormstore.Store) (uuid.UUID, bool) {
	sessionID := GetSystemSessionID()
	session, err := store.Get(r, sessionID)
	if err != nil {
		return uuid.Nil, false
	}

	userID, ok := session.Values["userID"].(uuid.UUID)
	return userID, ok
}

func GetUserOrgFromSession(r *http.Request, store *gormstore.Store) (uuid.UUID, bool) {
	if store == nil {
		log.Println("Session store is not initialized")
		return uuid.Nil, false
	}

	sessionID := GetSystemSessionID()
	session, err := store.Get(r, sessionID)
	if err != nil {
		log.Printf("Error retrieving session: %v", err)
		return uuid.Nil, false
	}

	orgID, ok := session.Values["organizationID"].(uuid.UUID)
	if !ok {
		log.Println("organizationID not found in session")
		return uuid.Nil, false
	}

	return orgID, true
}
