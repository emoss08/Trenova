package handlers

import (
	"backend/models"
	"backend/validation"
	"encoding/json"
	"net/http"
)

func TestEndpoint(w http.ResponseWriter, r *http.Request) {
	var data models.RequestData

	// Decode and validate
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fieldErrors, err := validation.ValidateStruct(data)
	if err != nil {
		// Handle non-validation errors (e.g., internal errors)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if fieldErrors != nil {
		respondWithError(w, http.StatusBadRequest, "error", "Validation error", fieldErrors)
		return
	}

	// Normal processing
	json.NewEncoder(w).Encode(map[string]string{"message": "Success"})
}

func respondWithError(w http.ResponseWriter, statusCode int, messageType, detail string, fieldErrors []models.FieldError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := models.APIErrorResponse{
		Type:    messageType,
		Message: detail,
		Errors:  fieldErrors,
	}

	json.NewEncoder(w).Encode(resp)
}
