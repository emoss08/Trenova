package validationframework

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/shared/pulid"
)

type ReferenceChecker interface {
	CheckReference(ctx context.Context, req *ReferenceRequest) (bool, error)
}

type ReferenceRequest struct {
	TableName      string
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	ID             pulid.ID
}

type CustomReferenceCheckFunc func(ctx context.Context, orgID, buID, refID pulid.ID) (bool, error)

// NewUSStateReferenceCheck builds the shared existence check for the global
// us_states table (which carries no tenant columns).
type ReferenceFieldConfig[T TenantedEntity] struct {
	FieldName   string
	TableName   string
	Message     string
	Optional    bool
	GetID       func(T) pulid.ID
	CustomCheck CustomReferenceCheckFunc
}
