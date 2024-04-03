package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"reflect"
	"testing"
)

func TestChargeTypeOps_CreateChargeType(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx           context.Context
		newChargeType ent.ChargeType
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.ChargeType
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ChargeTypeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.CreateChargeType(tt.args.ctx, tt.args.newChargeType)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateChargeType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateChargeType() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChargeTypeOps_GetChargeTypes(t *testing.T) {
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
		want    []*ent.ChargeType
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ChargeTypeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, got1, err := r.GetChargeTypes(tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetChargeTypes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetChargeTypes() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetChargeTypes() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestChargeTypeOps_UpdateChargeType(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx    context.Context
		entity ent.ChargeType
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.ChargeType
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ChargeTypeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.UpdateChargeType(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateChargeType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateChargeType() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewChargeTypeOps(t *testing.T) {
	tests := []struct {
		name string
		want *ChargeTypeOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewChargeTypeOps(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewChargeTypeOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
