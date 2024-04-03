package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"reflect"
	"testing"
)

func TestLoginOps_AuthenticateUser(t *testing.T) {
	type fields struct {
		ctx    context.Context
		client *ent.Client
	}
	type args struct {
		username string
		password string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.User
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &LoginOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			got, err := r.AuthenticateUser(tt.args.username, tt.args.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthenticateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AuthenticateUser() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewLoginOps(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want *LoginOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewLoginOps(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewLoginOps() = %v, want %v", got, tt.want)
			}
		})
	}
}
