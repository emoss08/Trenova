package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"reflect"
	"testing"
)

func TestNewTableChangeAlertOps(t *testing.T) {
	tests := []struct {
		name string
		want *TableChangeAlertOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTableChangeAlertOps(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTableChangeAlertOps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTableChangeAlertOps_CreateTableChangeAlert(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx       context.Context
		newEntity ent.TableChangeAlert
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.TableChangeAlert
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &TableChangeAlertOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.CreateTableChangeAlert(tt.args.ctx, tt.args.newEntity)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTableChangeAlert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateTableChangeAlert() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTableChangeAlertOps_GetTableChangeAlerts(t *testing.T) {
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
		want    []*ent.TableChangeAlert
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &TableChangeAlertOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, got1, err := r.GetTableChangeAlerts(tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTableChangeAlerts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTableChangeAlerts() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetTableChangeAlerts() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestTableChangeAlertOps_GetTableNames(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []TableName
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &TableChangeAlertOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, got1, err := r.GetTableNames(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTableNames() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTableNames() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetTableNames() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestTableChangeAlertOps_GetTopicNames(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		want    []TopicName
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &TableChangeAlertOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, got1, err := r.GetTopicNames()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTopicNames() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTopicNames() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetTopicNames() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestTableChangeAlertOps_UpdateTableChangeAlert(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx    context.Context
		entity ent.TableChangeAlert
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.TableChangeAlert
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &TableChangeAlertOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.UpdateTableChangeAlert(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateTableChangeAlert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateTableChangeAlert() got = %v, want %v", got, tt.want)
			}
		})
	}
}
