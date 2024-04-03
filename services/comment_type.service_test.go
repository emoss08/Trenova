package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"reflect"
	"testing"
)

func TestCommentTypeOps_CreateCommentType(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx       context.Context
		newEntity ent.CommentType
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.CommentType
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &CommentTypeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.CreateCommentType(tt.args.ctx, tt.args.newEntity)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateCommentType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateCommentType() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommentTypeOps_GetCommentTypes(t *testing.T) {
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
		want    []*ent.CommentType
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &CommentTypeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, got1, err := r.GetCommentTypes(tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCommentTypes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCommentTypes() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetCommentTypes() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestCommentTypeOps_UpdateCommentType(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx    context.Context
		entity ent.CommentType
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.CommentType
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &CommentTypeOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.UpdateCommentType(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateCommentType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateCommentType() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewCommentTypeOps(t *testing.T) {
	tests := []struct {
		name string
		want *CommentTypeOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewCommentTypeOps(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCommentTypeOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
