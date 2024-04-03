package controllers

import (
	"net/http"
	"testing"
)

func TestCreateDocumentClassification(t *testing.T) {
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
			CreateDocumentClassification(tt.args.w, tt.args.r)
		})
	}
}

func TestGetDocumentClassifications(t *testing.T) {
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
			GetDocumentClassifications(tt.args.w, tt.args.r)
		})
	}
}

func TestUpdateDocumentClassification(t *testing.T) {
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
			UpdateDocumentClassification(tt.args.w, tt.args.r)
		})
	}
}
