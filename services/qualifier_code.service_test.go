package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"reflect"
	"testing"
)

func TestNewQualifierCodeOps(t *testing.T) {
	tests := []struct {
		name string
		want *QualifierCodeOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewQualifierCodeOps(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewQualifierCodeOps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQualifierCodeOps_CreateQualifierCode(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx       context.Context
		newEntity ent.QualifierCode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.QualifierCode
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &QualifierCodeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.CreateQualifierCode(tt.args.ctx, tt.args.newEntity)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateQualifierCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateQualifierCode() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQualifierCodeOps_GetQualifierCodes(t *testing.T) {
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
		want    []*ent.QualifierCode
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &QualifierCodeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, got1, err := r.GetQualifierCodes(tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetQualifierCodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetQualifierCodes() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetQualifierCodes() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestQualifierCodeOps_UpdateQualifierCode(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx    context.Context
		entity ent.QualifierCode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.QualifierCode
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &QualifierCodeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.UpdateQualifierCode(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateQualifierCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateQualifierCode() got = %v, want %v", got, tt.want)
			}
		})
	}
}
