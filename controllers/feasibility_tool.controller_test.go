package controllers

import (
	"net/http"
	"testing"
)

func TestGetFeasibilityToolControl(t *testing.T) {
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
			GetFeasibilityToolControl(tt.args.w, tt.args.r)
		})
	}
}

func TestUpdateFeasibilityToolControl(t *testing.T) {
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
			UpdateFeasibilityToolControl(tt.args.w, tt.args.r)
		})
	}
}
