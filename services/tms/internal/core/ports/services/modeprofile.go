package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ResolveModeProfileRequest struct {
	TenantInfo     pagination.TenantInfo
	CustomerID     pulid.ID
	ServiceTypeID  pulid.ID
	ShipmentTypeID pulid.ID
	TractorTypeID  pulid.ID
	TrailerTypeID  pulid.ID
	At             int64
}

type RecordDeviationsRequest struct {
	TenantInfo   pagination.TenantInfo
	ResourceType string
	ResourceID   pulid.ID
	Policy       *modeprofile.ResolvedPolicy
	Advisories   []*errortypes.Advisory
}

type AcknowledgeDeviationServiceRequest struct {
	TenantInfo       pagination.TenantInfo
	DeviationID      pulid.ID
	AcknowledgedByID pulid.ID
	Reason           string
}

type ModeProfileService interface {
	Resolve(
		ctx context.Context,
		req *ResolveModeProfileRequest,
	) (*modeprofile.ResolvedPolicy, error)

	RecordDeviations(
		ctx context.Context,
		req *RecordDeviationsRequest,
	) error

	Acknowledge(
		ctx context.Context,
		req *AcknowledgeDeviationServiceRequest,
	) (*modeprofile.Deviation, error)

	ListDeviations(
		ctx context.Context,
		req *repositories.ListDeviationsRequest,
	) (*pagination.ListResult[*modeprofile.Deviation], error)

	Ledger(
		ctx context.Context,
		req *repositories.DeviationLedgerRequest,
	) ([]*repositories.DeviationLedgerEntry, error)
}
