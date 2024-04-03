package controllers

import (
	"net/http"
	"testing"
)

func TestCreateUserFavorite(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CreateUserFavorite(tt.args.w, tt.args.r)
		})
	}
}

func TestDeleteUserFavorite(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DeleteUserFavorite(tt.args.w, tt.args.r)
		})
	}
}

func TestGetUserFavorites(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GetUserFavorites(tt.args.w, tt.args.r)
		})
	}
}
