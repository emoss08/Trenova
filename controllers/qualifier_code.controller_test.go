package controllers

import (
	"net/http"
	"testing"
)

func TestCreateQualifierCode(t *testing.T) {
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
			CreateQualifierCode(tt.args.w, tt.args.r)
		})
	}
}

func TestGetQualifierCodes(t *testing.T) {
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
			GetQualifierCodes(tt.args.w, tt.args.r)
		})
	}
}

func TestUpdateQualifierCode(t *testing.T) {
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
			UpdateQualifierCode(tt.args.w, tt.args.r)
		})
	}
}
