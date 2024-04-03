package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"reflect"
	"testing"
)

func TestNewOrganizationOps(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want *OrganizatinOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewOrganizationOps(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewOrganizationOps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrganizatinOps_GetUserOrganization(t *testing.T) {
	type fields struct {
		ctx    context.Context
		client *ent.Client
	}
	type args struct {
		buID  uuid.UUID
		orgID uuid.UUID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.Organization
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &OrganizatinOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			got, err := r.GetUserOrganization(tt.args.buID, tt.args.orgID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserOrganization() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetUserOrganization() got = %v, want %v", got, tt.want)
			}
		})
	}
}
