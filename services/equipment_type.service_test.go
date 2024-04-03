package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"reflect"
	"testing"
)

func TestEquipmentTypeOps_CreateEquipmentType(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx       context.Context
		newEntity ent.EquipmentType
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.EquipmentType
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EquipmentTypeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.CreateEquipmentType(tt.args.ctx, tt.args.newEntity)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateEquipmentType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateEquipmentType() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEquipmentTypeOps_GetEquipmentTypes(t *testing.T) {
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
		want    []*ent.EquipmentType
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EquipmentTypeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, got1, err := r.GetEquipmentTypes(tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetEquipmentTypes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEquipmentTypes() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetEquipmentTypes() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestEquipmentTypeOps_UpdateEquipmentType(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx    context.Context
		entity ent.EquipmentType
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.EquipmentType
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EquipmentTypeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.UpdateEquipmentType(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateEquipmentType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateEquipmentType() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewEquipmentTypeOps(t *testing.T) {
	tests := []struct {
		name string
		want *EquipmentTypeOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewEquipmentTypeOps(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEquipmentTypeOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
