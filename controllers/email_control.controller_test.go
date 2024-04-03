package controllers

import (
	"net/http"
	"testing"
)

func TestGetEmailControl(t *testing.T) {
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
			GetEmailControl(tt.args.w, tt.args.r)
		})
	}
}

func TestUpdateEmailControl(t *testing.T) {
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
			UpdateEmailControl(tt.args.w, tt.args.r)
		})
	}
}
