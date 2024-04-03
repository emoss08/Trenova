package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"reflect"
	"testing"
)

func TestHazardousMaterialSegregationOps_CreateHazmatSegRule(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx       context.Context
		newEntity ent.HazardousMaterialSegregation
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.HazardousMaterialSegregation
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &HazardousMaterialSegregationOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.CreateHazmatSegRule(tt.args.ctx, tt.args.newEntity)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateHazmatSegRule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateHazmatSegRule() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHazardousMaterialSegregationOps_GetHazmatSegRules(t *testing.T) {
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
		want    []*ent.HazardousMaterialSegregation
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &HazardousMaterialSegregationOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, got1, err := r.GetHazmatSegRules(tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetHazmatSegRules() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetHazmatSegRules() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetHazmatSegRules() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestHazardousMaterialSegregationOps_UpdateHazmatSegRule(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx    context.Context
		entity ent.HazardousMaterialSegregation
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.HazardousMaterialSegregation
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &HazardousMaterialSegregationOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.UpdateHazmatSegRule(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateHazmatSegRule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateHazmatSegRule() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewHazardousMaterialSegregationOps(t *testing.T) {
	tests := []struct {
		name string
		want *HazardousMaterialSegregationOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewHazardousMaterialSegregationOps(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHazardousMaterialSegregationOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
