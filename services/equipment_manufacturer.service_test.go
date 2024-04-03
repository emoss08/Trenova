package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"reflect"
	"testing"
)

func TestEquipmentManufactuerOps_CreateEquipmentManufacturer(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx       context.Context
		newEntity ent.EquipmentManufactuer
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.EquipmentManufactuer
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EquipmentManufactuerOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.CreateEquipmentManufacturer(tt.args.ctx, tt.args.newEntity)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateEquipmentManufacturer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateEquipmentManufacturer() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEquipmentManufactuerOps_GetEquipmentManufacturers(t *testing.T) {
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
		want    []*ent.EquipmentManufactuer
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EquipmentManufactuerOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, got1, err := r.GetEquipmentManufacturers(tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetEquipmentManufacturers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEquipmentManufacturers() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetEquipmentManufacturers() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestEquipmentManufactuerOps_UpdateEquipmentManufacturer(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx    context.Context
		entity ent.EquipmentManufactuer
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.EquipmentManufactuer
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EquipmentManufactuerOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.UpdateEquipmentManufacturer(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateEquipmentManufacturer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateEquipmentManufacturer() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewEquipmentManufactuerOps(t *testing.T) {
	tests := []struct {
		name string
		want *EquipmentManufactuerOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewEquipmentManufactuerOps(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEquipmentManufactuerOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
