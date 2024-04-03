package controllers

import (
	"net/http"
	"testing"
)

func TestCreateHazmatSegRule(t *testing.T) {
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
			CreateHazmatSegRule(tt.args.w, tt.args.r)
		})
	}
}

func TestGetHazmatSegRules(t *testing.T) {
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
			GetHazmatSegRules(tt.args.w, tt.args.r)
		})
	}
}

func TestUpdateHazmatSegRule(t *testing.T) {
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
			UpdateHazmatSegRule(tt.args.w, tt.args.r)
		})
	}
}
