package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"reflect"
	"testing"
)

func TestFleetCodeOps_CreateFleetCode(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx       context.Context
		newEntity ent.FleetCode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.FleetCode
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &FleetCodeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.CreateFleetCode(tt.args.ctx, tt.args.newEntity)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateFleetCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateFleetCode() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFleetCodeOps_GetFleetCodes(t *testing.T) {
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
		want    []*ent.FleetCode
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &FleetCodeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, got1, err := r.GetFleetCodes(tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFleetCodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetFleetCodes() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetFleetCodes() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestFleetCodeOps_UpdateFleetCode(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx    context.Context
		entity ent.FleetCode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.FleetCode
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &FleetCodeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.UpdateFleetCode(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateFleetCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateFleetCode() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewFleetCodeOps(t *testing.T) {
	tests := []struct {
		name string
		want *FleetCodeOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewFleetCodeOps(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewFleetCodeOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
