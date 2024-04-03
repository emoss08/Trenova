package controllers

import (
	"net/http"
	"testing"
)

func TestCreateHazardousMaterial(t *testing.T) {
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
			CreateHazardousMaterial(tt.args.w, tt.args.r)
		})
	}
}

func TestGetHazardousMaterial(t *testing.T) {
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
			GetHazardousMaterial(tt.args.w, tt.args.r)
		})
	}
}

func TestUpdateHazardousMaterial(t *testing.T) {
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
			UpdateHazardousMaterial(tt.args.w, tt.args.r)
		})
	}
}
