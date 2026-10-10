package repositories

import (
	"context"
	"time"
)

type WeatherAlertFeedState struct {
	ETag          string `json:"etag,omitempty"`
	LastModified  string `json:"lastModified,omitempty"`
	TenantsDigest string `json:"tenantsDigest"`
}

type WeatherAlertFeedStateStore interface {
	Get(ctx context.Context) (*WeatherAlertFeedState, error)
	Save(ctx context.Context, state *WeatherAlertFeedState, ttl time.Duration) error
}
