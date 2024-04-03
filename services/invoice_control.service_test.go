package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"reflect"
	"testing"
)

func TestInvoiceControlOps_GetInvoiceControlByOrgID(t *testing.T) {
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
		want    *ent.InvoiceControl
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &InvoiceControlOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			got, err := r.GetInvoiceControlByOrgID(tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInvoiceControlByOrgID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetInvoiceControlByOrgID() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvoiceControlOps_UpdateInvoiceControl(t *testing.T) {
	type fields struct {
		ctx    context.Context
		client *ent.Client
	}
	type args struct {
		ic ent.InvoiceControl
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.InvoiceControl
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &InvoiceControlOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			got, err := r.UpdateInvoiceControl(tt.args.ic)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateInvoiceControl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateInvoiceControl() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewInvoiceControlOps(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want *InvoiceControlOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewInvoiceControlOps(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewInvoiceControlOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
