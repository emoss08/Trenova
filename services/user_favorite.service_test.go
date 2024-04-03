package services

import (
	"context"
	"github.com/emoss08/trenova/ent"
	"github.com/google/uuid"
	"reflect"
	"testing"
)

func TestNewUserFavoriteOps(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want *UserFavoriteOps
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewUserFavoriteOps(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUserFavoriteOps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserFavoriteOps_GetUserFavorites(t *testing.T) {
	type fields struct {
		ctx    context.Context
		client *ent.Client
	}
	type args struct {
		userID uuid.UUID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*ent.UserFavorite
		want1   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &UserFavoriteOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			got, got1, err := r.GetUserFavorites(tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserFavorites() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetUserFavorites() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetUserFavorites() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestUserFavoriteOps_UserFavoriteCreate(t *testing.T) {
	type fields struct {
		ctx    context.Context
		client *ent.Client
	}
	type args struct {
		userFavorite ent.UserFavorite
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ent.UserFavorite
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &UserFavoriteOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			got, err := r.UserFavoriteCreate(tt.args.userFavorite)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserFavoriteCreate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UserFavoriteCreate() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserFavoriteOps_UserFavoriteDelete(t *testing.T) {
	type fields struct {
		ctx    context.Context
		client *ent.Client
	}
	type args struct {
		userID   uuid.UUID
		pageLink string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &UserFavoriteOps{
				ctx:    tt.fields.ctx,
				client: tt.fields.client,
			}
			if err := r.UserFavoriteDelete(tt.args.userID, tt.args.pageLink); (err != nil) != tt.wantErr {
				t.Errorf("UserFavoriteDelete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
