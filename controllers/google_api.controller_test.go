package controllers

import (
	"net/http"
	"testing"
)

func TestGetGoogleAPI(t *testing.T) {
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
			GetGoogleAPI(tt.args.w, tt.args.r)
		})
	}
}

func TestUpdateGoogleAPI(t *testing.T) {
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
			UpdateGoogleAPI(tt.args.w, tt.args.r)
		})
	}
}
