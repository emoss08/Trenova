package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"reflect"
	"testing"
)

func TestHazardousMaterialOps_CreateHazardousMaterial(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx       context.Context
		newEntity ent.HazardousMaterial
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.HazardousMaterial
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &HazardousMaterialOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.CreateHazardousMaterial(tt.args.ctx, tt.args.newEntity)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateHazardousMaterial() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateHazardousMaterial() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHazardousMaterialOps_GetHazardousMaterials(t *testing.T) {
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
		want    []*ent.HazardousMaterial
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &HazardousMaterialOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, got1, err := r.GetHazardousMaterials(tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetHazardousMaterials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetHazardousMaterials() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetHazardousMaterials() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestHazardousMaterialOps_UpdateHazardousMaterial(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx    context.Context
		entity ent.HazardousMaterial
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.HazardousMaterial
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &HazardousMaterialOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.UpdateHazardousMaterial(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateHazardousMaterial() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateHazardousMaterial() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewHazardousMaterialOps(t *testing.T) {
	tests := []struct {
		name string
		want *HazardousMaterialOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewHazardousMaterialOps(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHazardousMaterialOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
