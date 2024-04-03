package controllers

import (
	"net/http"
	"testing"
)

func TestGetBillingControl(t *testing.T) {
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
			GetBillingControl(tt.args.w, tt.args.r)
		})
	}
}

func TestUpdateBillingControl(t *testing.T) {
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
			UpdateBillingControl(tt.args.w, tt.args.r)
		})
	}
}
