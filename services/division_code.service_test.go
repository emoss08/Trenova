package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"reflect"
	"testing"
)

func TestDivisionCodeOps_CreateDivisionCode(t *testing.T) {
	type fields struct {
		logger *logrus.Logger
		client *ent.Client
	}
	type args struct {
		ctx       context.Context
		newEntity ent.DivisionCode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.DivisionCode
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &DivisionCodeOps{
				logger: tt.fields.logger,
				client: tt.fields.client,
			}
			got, err := r.CreateDivisionCode(tt.args.ctx, tt.args.newEntity)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateDivisionCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateDivisionCode() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDivisionCodeOps_GetDivisionCodes(t *testing.T) {
	type fields struct {
		logger *logrus.Logger
		client *ent.Client
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
		want    []*ent.DivisionCode
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &DivisionCodeOps{
				logger: tt.fields.logger,
				client: tt.fields.client,
			}
			got, got1, err := r.GetDivisionCodes(tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDivisionCodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDivisionCodes() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetDivisionCodes() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestDivisionCodeOps_UpdateDivisionCode(t *testing.T) {
	type fields struct {
		logger *logrus.Logger
		client *ent.Client
	}
	type args struct {
		ctx    context.Context
		entity ent.DivisionCode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.DivisionCode
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &DivisionCodeOps{
				logger: tt.fields.logger,
				client: tt.fields.client,
			}
			got, err := r.UpdateDivisionCode(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateDivisionCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateDivisionCode() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewDivisionCodeOps(t *testing.T) {
	tests := []struct {
		name string
		want *DivisionCodeOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDivisionCodeOps(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDivisionCodeOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
