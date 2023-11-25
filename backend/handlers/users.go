package handlers

import (
	"backend/service"
	"encoding/json"
	"net/http"
)

func CreateUserHandler(s *service.ServiceContainer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req service.CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := s.UserService.CreateUser(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(req)
	}
}

func GetAllUsersHandler(s *service.ServiceContainer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := s.UserService.GetAllUsers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(users)
	}
}
