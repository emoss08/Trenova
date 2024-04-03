package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"reflect"
	"testing"
)

func TestNewShipmentTypeOps(t *testing.T) {
	tests := []struct {
		name string
		want *ShipmentTypeOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewShipmentTypeOps(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewShipmentTypeOps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShipmentTypeOps_CreateShipmentType(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx       context.Context
		newEntity ent.ShipmentType
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.ShipmentType
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ShipmentTypeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.CreateShipmentType(tt.args.ctx, tt.args.newEntity)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateShipmentType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateShipmentType() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShipmentTypeOps_GetShipmentTypes(t *testing.T) {
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
		want    []*ent.ShipmentType
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ShipmentTypeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, got1, err := r.GetShipmentTypes(tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetShipmentTypes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetShipmentTypes() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetShipmentTypes() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestShipmentTypeOps_UpdateShipmentType(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx    context.Context
		entity ent.ShipmentType
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.ShipmentType
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ShipmentTypeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.UpdateShipmentType(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateShipmentType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateShipmentType() got = %v, want %v", got, tt.want)
			}
		})
	}
}
