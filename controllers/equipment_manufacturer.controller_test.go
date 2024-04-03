package controllers

import (
	"net/http"
	"testing"
)

func TestCreateEquipmentManufacturer(t *testing.T) {
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
			CreateEquipmentManufacturer(tt.args.w, tt.args.r)
		})
	}
}

func TestGetEquipmentManufacturer(t *testing.T) {
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
			GetEquipmentManufacturer(tt.args.w, tt.args.r)
		})
	}
}

func TestUpdateEquipmentManfacturer(t *testing.T) {
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
			UpdateEquipmentManfacturer(tt.args.w, tt.args.r)
		})
	}
}
