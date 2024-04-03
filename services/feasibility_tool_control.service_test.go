package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"reflect"
	"testing"
)

func TestFeasibilityControlOps_GetFeasibilityToolControl(t *testing.T) {
	type fields struct {
		ctx    context.Context
		client *ent.Client
	}
	type args struct {
		orgID uuid.UUID
		buID  uuid.UUID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.FeasibilityToolControl
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &FeasibilityControlOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			got, err := r.GetFeasibilityToolControl(tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFeasibilityToolControl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetFeasibilityToolControl() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFeasibilityControlOps_UpdateFeasibilityToolControl(t *testing.T) {
	type fields struct {
		ctx    context.Context
		client *ent.Client
	}
	type args struct {
		ftc ent.FeasibilityToolControl
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.FeasibilityToolControl
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &FeasibilityControlOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			got, err := r.UpdateFeasibilityToolControl(tt.args.ftc)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateFeasibilityToolControl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateFeasibilityToolControl() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewFeasibilityControlOps(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want *FeasibilityControlOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewFeasibilityControlOps(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewFeasibilityControlOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
