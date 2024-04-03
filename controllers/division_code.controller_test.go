package controllers

import (
	"net/http"
	"testing"
)

func TestCreateDivisionCode(t *testing.T) {
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
			CreateDivisionCode(tt.args.w, tt.args.r)
		})
	}
}

func TestGetDivisionCodes(t *testing.T) {
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
			GetDivisionCodes(tt.args.w, tt.args.r)
		})
	}
}

func TestUpdateDivisionCode(t *testing.T) {
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
			UpdateDivisionCode(tt.args.w, tt.args.r)
		})
	}
}
