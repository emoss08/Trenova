package controllers

import (
	"net/http"
	"testing"
)

func TestCreateEquipmentType(t *testing.T) {
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
			CreateEquipmentType(tt.args.w, tt.args.r)
		})
	}
}

func TestGetEquipmentTypes(t *testing.T) {
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
			GetEquipmentTypes(tt.args.w, tt.args.r)
		})
	}
}

func TestUpdateEquipmentType(t *testing.T) {
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
			UpdateEquipmentType(tt.args.w, tt.args.r)
		})
	}
}
