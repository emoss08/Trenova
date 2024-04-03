package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"reflect"
	"testing"
)

func TestBillingControlOps_GetBillingControl(t *testing.T) {
	type fields struct {
		client *ent.Client
	}
	type args struct {
		ctx   context.Context
		orgID uuid.UUID
		buID  uuid.UUID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.BillingControl
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &BillingControlOps{
				client: tt.fields.client,
			}
			got, err := r.GetBillingControl(tt.args.ctx, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBillingControl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBillingControl() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBillingControlOps_UpdateBillingControl(t *testing.T) {
	type fields struct {
		client *ent.Client
	}
	type args struct {
		ctx context.Context
		bc  ent.BillingControl
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.BillingControl
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &BillingControlOps{
				client: tt.fields.client,
			}
			got, err := r.UpdateBillingControl(tt.args.ctx, tt.args.bc)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateBillingControl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateBillingControl() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewBillingControlOps(t *testing.T) {
	tests := []struct {
		name string
		want *BillingControlOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewBillingControlOps(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBillingControlOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
