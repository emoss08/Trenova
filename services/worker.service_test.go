package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"reflect"
	"testing"
)

func TestNewWorkerOps(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want *WorkerOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewWorkerOps(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewWorkerOps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorkerOps_CreateWorker(t *testing.T) {
	type fields struct {
		ctx    context.Context
		client *ent.Client
	}
	type args struct {
		entity WorkerRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.Worker
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &WorkerOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			got, err := r.CreateWorker(tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateWorker() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateWorker() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorkerOps_GetWorkers(t *testing.T) {
	type fields struct {
		ctx    context.Context
		client *ent.Client
	}
	type args struct {
		limit  int
		offset int
		orgID  uuid.UUID
		buID   uuid.UUID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*ent.Worker
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &WorkerOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			got, got1, err := r.GetWorkers(tt.args.limit, tt.args.offset, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetWorkers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetWorkers() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetWorkers() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestWorkerOps_UpdateWorker(t *testing.T) {
	type fields struct {
		ctx    context.Context
		client *ent.Client
	}
	type args struct {
		entity WorkerUpdateRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.Worker
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &WorkerOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			got, err := r.UpdateWorker(tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateWorker() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateWorker() got = %v, want %v", got, tt.want)
			}
		})
	}
}
