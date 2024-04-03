package controllers

import (
	"net/http"
	"testing"
)

func TestGetInvoiceControl(t *testing.T) {
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
			GetInvoiceControl(tt.args.w, tt.args.r)
		})
	}
}

func TestUpdateInvoiceControl(t *testing.T) {
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
			UpdateInvoiceControl(tt.args.w, tt.args.r)
		})
	}
}
