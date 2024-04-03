package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"reflect"
	"testing"
)

func TestCommodityOps_CreateCommodity(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx       context.Context
		newEntity ent.Commodity
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.Commodity
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &CommodityOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.CreateCommodity(tt.args.ctx, tt.args.newEntity)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateCommodity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateCommodity() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommodityOps_GetCommodities(t *testing.T) {
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
		want    []*ent.Commodity
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &CommodityOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, got1, err := r.GetCommodities(tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCommodities() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCommodities() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetCommodities() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestCommodityOps_UpdateCommodity(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx    context.Context
		entity ent.Commodity
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.Commodity
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &CommodityOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.UpdateCommodity(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateCommodity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateCommodity() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewCommodityOps(t *testing.T) {
	tests := []struct {
		name string
		want *CommodityOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewCommodityOps(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCommodityOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
