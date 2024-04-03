package controllers

import (
	"net/http"
	"testing"
)

func TestCreateAccessorialCharge(t *testing.T) {
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
			CreateAccessorialCharge(tt.args.w, tt.args.r)
		})
	}
}

func TestGetAccessorialCharge(t *testing.T) {
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
			GetAccessorialCharge(tt.args.w, tt.args.r)
		})
	}
}

func TestUpdateAccessorialCharge(t *testing.T) {
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
			UpdateAccessorialCharge(tt.args.w, tt.args.r)
		})
	}
}
