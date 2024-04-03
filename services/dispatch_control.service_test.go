package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"reflect"
	"testing"
)

func TestDispatchControlOps_GetDispatchControl(t *testing.T) {
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
		want    *ent.DispatchControl
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &DispatchControlOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			got, err := r.GetDispatchControl(tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDispatchControl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDispatchControl() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDispatchControlOps_UpdateDispatchControl(t *testing.T) {
	type fields struct {
		ctx    context.Context
		client *ent.Client
	}
	type args struct {
		dc ent.DispatchControl
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.DispatchControl
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &DispatchControlOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			got, err := r.UpdateDispatchControl(tt.args.dc)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateDispatchControl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateDispatchControl() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewDispatchControlOps(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want *DispatchControlOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDispatchControlOps(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDispatchControlOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
