package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"reflect"
	"testing"
)

func TestNewShipmentControlOps(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want *ShipmentControlOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewShipmentControlOps(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewShipmentControlOps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShipmentControlOps_GetShipmentControl(t *testing.T) {
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
		want    *ent.ShipmentControl
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ShipmentControlOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			got, err := r.GetShipmentControl(tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetShipmentControl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetShipmentControl() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShipmentControlOps_UpdateShipmentControl(t *testing.T) {
	type fields struct {
		ctx    context.Context
		client *ent.Client
	}
	type args struct {
		sc ent.ShipmentControl
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.ShipmentControl
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ShipmentControlOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			got, err := r.UpdateShipmentControl(tt.args.sc)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateShipmentControl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateShipmentControl() got = %v, want %v", got, tt.want)
			}
		})
	}
}
