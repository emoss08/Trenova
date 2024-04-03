package controllers

import (
	"net/http"
	"testing"
)

func TestCreateGeneralLedgerAccount(t *testing.T) {
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
			CreateGeneralLedgerAccount(tt.args.w, tt.args.r)
		})
	}
}

func TestGetGeneralLedgerAccounts(t *testing.T) {
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
			GetGeneralLedgerAccounts(tt.args.w, tt.args.r)
		})
	}
}

func TestUpdateGeneralLedgerAccount(t *testing.T) {
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
			UpdateGeneralLedgerAccount(tt.args.w, tt.args.r)
		})
	}
}
