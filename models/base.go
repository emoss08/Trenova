package models

import (
	"time"

	"github.com/google/uuid"
)

// Model defines an interface with common methods that all db models should have.
type Model interface {
	SetOrgID(uuid.UUID)
	SetBuID(uuid.UUID)
	GetId() uuid.UUID
	GetCreated() time.Time
	GetUpdated() time.Time
	RefreshCreated()
	RefreshUpdated()
}

type BaseModel struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid();"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (m *BaseModel) GetId() uuid.UUID {
	return m.ID
}
