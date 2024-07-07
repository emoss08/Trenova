package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type AuditLog struct {
	bun.BaseModel `bun:"table:audit_logs,alias:al"`
	ID            uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()"`
	EntityType    string          `bun:"type:varchar(50),notnull"`
	EntityID      uuid.UUID       `bun:"type:uuid,notnull"`
	Action        string          `bun:"type:varchar(50),notnull"`
	ChangedFields json.RawMessage `bun:"type:jsonb"`
	UserID        uuid.UUID       `bun:"type:uuid,notnull"`
	Timestamp     time.Time       `bun:",nullzero,notnull,default:current_timestamp"`
}
