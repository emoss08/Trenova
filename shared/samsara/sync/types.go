package samsync

import "context"

type Entity string

const (
	EntityAddresses Entity = "addresses"
	EntityDrivers   Entity = "drivers"
	EntityAssets    Entity = "assets"
	EntityRoutes    Entity = "routes"
	EntityForms     Entity = "forms"
)

type Direction string

const (
	DirectionPull Direction = "pull"
	DirectionPush Direction = "push"
)

type Mapping struct {
	TenantID    string
	Entity      Entity
	TMSID       string
	SamsaraID   string
	ExternalIDs map[string]string
}

type Cursor struct {
	TenantID string
	Entity   Entity
	Value    string
}

type Event struct {
	TenantID    string
	Entity      Entity
	EventType   string
	SamsaraID   string
	OccurredAt  string
	RawPayload  map[string]any
	ExternalIDs map[string]string
}

type MappingStore interface {
	GetByTMSID(ctx context.Context, tenantID string, entity Entity, tmsID string) (*Mapping, error)
	GetBySamsaraID(
		ctx context.Context,
		tenantID string,
		entity Entity,
		samsaraID string,
	) (*Mapping, error)
	Upsert(ctx context.Context, mapping Mapping) error
}

type CursorStore interface {
	Get(ctx context.Context, tenantID string, entity Entity) (*Cursor, error)
	Set(ctx context.Context, cursor Cursor) error
}

type ConflictResolver interface {
	ShouldPushToSamsara(
		ctx context.Context,
		tenantID string,
		entity Entity,
		tmsRecord map[string]any,
	) (bool, error)
	ShouldPullFromSamsara(
		ctx context.Context,
		tenantID string,
		entity Entity,
		samsaraRecord map[string]any,
	) (bool, error)
}

type SyncJob interface {
	Name() string
	Run(ctx context.Context, tenantID string) error
}

type WebhookIngestor interface {
	Handle(ctx context.Context, event Event) error
}

type KafkaIngestor interface {
	Handle(ctx context.Context, event Event) error
}
