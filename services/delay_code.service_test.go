package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"reflect"
	"testing"
)

func TestDelayCodeOps_CreateDelayCode(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx       context.Context
		newEntity ent.DelayCode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.DelayCode
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &DelayCodeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.CreateDelayCode(tt.args.ctx, tt.args.newEntity)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateDelayCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateDelayCode() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDelayCodeOps_GetDelayCodes(t *testing.T) {
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
		want    []*ent.DelayCode
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &DelayCodeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, got1, err := r.GetDelayCodes(tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDelayCodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDelayCodes() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetDelayCodes() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestDelayCodeOps_UpdateDelayCode(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx    context.Context
		entity ent.DelayCode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.DelayCode
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &DelayCodeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.UpdateDelayCode(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateDelayCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateDelayCode() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewDelayCodeOps(t *testing.T) {
	tests := []struct {
		name string
		want *DelayCodeOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDelayCodeOps(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDelayCodeOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
