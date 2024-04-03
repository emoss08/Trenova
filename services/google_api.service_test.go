package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"reflect"
	"testing"
)

func TestGoogleAPIOps_GetGoogleAPI(t *testing.T) {
	type fields struct {
		ctx    context.Context
		client *ent.Client
	}
	type args struct {
		orgID uuid.UUID
		buID  uuid.UUID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.GoogleApi
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &GoogleAPIOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			got, err := r.GetGoogleAPI(tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGoogleAPI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetGoogleAPI() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGoogleAPIOps_UpdateGoogleAPI(t *testing.T) {
	type fields struct {
		ctx    context.Context
		client *ent.Client
	}
	type args struct {
		googleAPI ent.GoogleApi
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.GoogleApi
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &GoogleAPIOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			got, err := r.UpdateGoogleAPI(tt.args.googleAPI)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateGoogleAPI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateGoogleAPI() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewGoogleAPIOps(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want *GoogleAPIOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewGoogleAPIOps(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewGoogleAPIOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
