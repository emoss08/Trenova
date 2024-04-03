package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"reflect"
	"testing"
)

func TestLocationCategoryOps_CreateLocationCategory(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx       context.Context
		newEntity ent.LocationCategory
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.LocationCategory
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &LocationCategoryOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.CreateLocationCategory(tt.args.ctx, tt.args.newEntity)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateLocationCategory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateLocationCategory() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLocationCategoryOps_GetLocationCategories(t *testing.T) {
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
		want    []*ent.LocationCategory
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &LocationCategoryOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, got1, err := r.GetLocationCategories(tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLocationCategories() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetLocationCategories() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetLocationCategories() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestLocationCategoryOps_UpdateLocationCategory(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx    context.Context
		entity ent.LocationCategory
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.LocationCategory
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &LocationCategoryOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.UpdateLocationCategory(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateLocationCategory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateLocationCategory() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewLocationCategoryOps(t *testing.T) {
	tests := []struct {
		name string
		want *LocationCategoryOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewLocationCategoryOps(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewLocationCategoryOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
