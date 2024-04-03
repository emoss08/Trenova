package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"reflect"
	"testing"
)

func TestEmailProfileOps_CreateEmailProfile(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx       context.Context
		newEntity ent.EmailProfile
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.EmailProfile
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EmailProfileOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.CreateEmailProfile(tt.args.ctx, tt.args.newEntity)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateEmailProfile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateEmailProfile() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmailProfileOps_GetEmailProfiles(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx    context.Context
		limit  int
		offset int
		orgID  uuid.UUID
		buID   uuid.UUID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*ent.EmailProfile
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EmailProfileOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, got1, err := r.GetEmailProfiles(tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetEmailProfiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEmailProfiles() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetEmailProfiles() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestEmailProfileOps_UpdateEmailProfile(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx    context.Context
		entity ent.EmailProfile
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.EmailProfile
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EmailProfileOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.UpdateEmailProfile(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateEmailProfile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateEmailProfile() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewEmailProfileOps(t *testing.T) {
	tests := []struct {
		name string
		want *EmailProfileOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewEmailProfileOps(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEmailProfileOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
