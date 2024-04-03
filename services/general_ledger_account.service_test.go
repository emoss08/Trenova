package services

import (
	"context"
	"reflect"
	"testing"

	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func TestGeneralLedgerAccountOps_CreateGeneralLedgerAccount(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx       context.Context
		newEntity GeneralLedgerAccountRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.GeneralLedgerAccount
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &GeneralLedgerAccountOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.CreateGeneralLedgerAccount(tt.args.ctx, tt.args.newEntity)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateGeneralLedgerAccount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateGeneralLedgerAccount() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeneralLedgerAccountOps_GetGeneralLedgerAccounts(t *testing.T) {
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
		want    []*ent.GeneralLedgerAccount
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &GeneralLedgerAccountOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, got1, err := r.GetGeneralLedgerAccounts(tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.orgID, tt.args.buID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGeneralLedgerAccounts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetGeneralLedgerAccounts() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetGeneralLedgerAccounts() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestGeneralLedgerAccountOps_UpdateGeneralLedgerAccount(t *testing.T) {
	type fields struct {
		client *ent.Client
		logger *logrus.Logger
	}
	type args struct {
		ctx    context.Context
		entity GeneralLedgerAccountUpdateRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.GeneralLedgerAccount
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &GeneralLedgerAccountOps{
				client: tt.fields.client,
				logger: tt.fields.logger,
			}
			got, err := r.UpdateGeneralLedgerAccount(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateGeneralLedgerAccount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateGeneralLedgerAccount() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewGeneralLedgerAccountOps(t *testing.T) {
	tests := []struct {
		name string
		want *GeneralLedgerAccountOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewGeneralLedgerAccountOps(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewGeneralLedgerAccountOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
