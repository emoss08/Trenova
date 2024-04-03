package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"reflect"
	"testing"
)

func TestDocumentClassificationOps_CreateDocumentClassification(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx       context.Context
		newEntity ent.DocumentClassification
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.DocumentClassification
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &DocumentClassificationOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.CreateDocumentClassification(tt.args.ctx, tt.args.newEntity)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateDocumentClassification() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateDocumentClassification() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDocumentClassificationOps_GetDocumentClassification(t *testing.T) {
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
		want    []*ent.DocumentClassification
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &DocumentClassificationOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, got1, err := r.GetDocumentClassification(tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDocumentClassification() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDocumentClassification() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetDocumentClassification() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestDocumentClassificationOps_UpdateDocumentClassification(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx    context.Context
		entity ent.DocumentClassification
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.DocumentClassification
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &DocumentClassificationOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.UpdateDocumentClassification(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateDocumentClassification() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateDocumentClassification() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewDocumentClassificationOps(t *testing.T) {
	tests := []struct {
		name string
		want *DocumentClassificationOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDocumentClassificationOps(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDocumentClassificationOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
