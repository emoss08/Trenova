package framework

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
)

type Validatable interface {
	Validate(multiErr *errortypes.MultiError)
}

type Identifiable interface {
	GetID() string
}

type Tableable interface {
	GetTableName() string
}

type Tenantable interface {
	GetOrganizationID() pulid.ID
	GetBusinessUnitID() pulid.ID
}

type ValidatableEntity interface {
	Validatable
	Identifiable
	Tableable
}

type TenantedEntity interface {
	ValidatableEntity
	Tenantable
}

type UniqueField struct {
	Name     string
	GetValue func() string
	Message  string // Optional custom message
}

type EntityMetadata interface {
	GetModelName() string
	GetUniqueFields() []UniqueField
}
