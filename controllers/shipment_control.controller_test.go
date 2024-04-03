package controllers

import (
	"net/http"
	"testing"
)

func TestGetShipmentControl(t *testing.T) {
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
			GetShipmentControl(tt.args.w, tt.args.r)
		})
	}
}

func TestUpdateShipmentControl(t *testing.T) {
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
			UpdateShipmentControl(tt.args.w, tt.args.r)
		})
	}
}
