package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/instancebootstrap"
	"github.com/emoss08/trenova/shared/pulid"
)

type InstanceBootstrapState struct {
	Record    *instancebootstrap.InstanceBootstrap
	UserCount int
}

type FinalizeInstanceBootstrapRequest struct {
	OrganizationID     pulid.ID
	BusinessUnitID     pulid.ID
	AdminUserID        pulid.ID
	Inputs             instancebootstrap.Inputs
	SystemUserPassword string
	Now                int64
}

type FinalizeInstanceBootstrapResult struct {
	Record            *instancebootstrap.InstanceBootstrap
	SystemUserID      pulid.ID
	SystemUserCreated bool
}

type InstanceBootstrapRepository interface {
	GetState(ctx context.Context) (*InstanceBootstrapState, error)
	Finalize(
		ctx context.Context,
		req *FinalizeInstanceBootstrapRequest,
	) (*FinalizeInstanceBootstrapResult, error)
}
