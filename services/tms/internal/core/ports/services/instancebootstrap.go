package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/instancebootstrap"
	"github.com/emoss08/trenova/shared/pulid"
)

type InstanceBootstrapStatus string

const (
	InstanceBootstrapReady     InstanceBootstrapStatus = "ready"
	InstanceBootstrapCreated   InstanceBootstrapStatus = "created"
	InstanceBootstrapCompleted InstanceBootstrapStatus = "completed"
)

type InstanceBootstrapPlan struct {
	Status InstanceBootstrapStatus
	Inputs instancebootstrap.Inputs
	Record *instancebootstrap.InstanceBootstrap
}

type InstanceBootstrapRequest struct {
	Inputs   instancebootstrap.Inputs
	Password string
}

type InstanceBootstrapResult struct {
	Status            InstanceBootstrapStatus
	OrganizationID    pulid.ID
	BusinessUnitID    pulid.ID
	AdminUserID       pulid.ID
	AdminUsername     string
	AdminEmail        string
	LoginSlug         string
	SystemUserCreated bool
	CompletedAt       int64
}

type InstanceBootstrapService interface {
	Plan(ctx context.Context, inputs *instancebootstrap.Inputs) (*InstanceBootstrapPlan, error)
	Bootstrap(
		ctx context.Context,
		req *InstanceBootstrapRequest,
	) (*InstanceBootstrapResult, error)
}
