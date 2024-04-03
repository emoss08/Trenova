package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"reflect"
	"testing"
)

func TestNewUsStateOps(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want *UsStateOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewUsStateOps(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUsStateOps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUsStateOps_GetUsStates(t *testing.T) {
	type fields struct {
		ctx    context.Context
		client *ent.Client
	}
	tests := []struct {
		name    string
		fields  fields
		want    []*ent.UsState
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &UsStateOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			got, err := r.GetUsStates()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUsStates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetUsStates() got = %v, want %v", got, tt.want)
			}
		})
	}
}
