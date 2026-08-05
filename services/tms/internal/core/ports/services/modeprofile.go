package services

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ErrNoModeProfileConfigured means the tenant has no active profile to resolve
// against. It is an ordinary outcome, not a fault: capability enforcement is
// opt-in, and a carrier that has configured no profile is not in error.
//
// It exists so a caller can tell that case apart from a lookup that failed.
// Collapsing both into a nil policy turns a database outage into "no rules
// apply", which silently disables every capability check.
var ErrNoModeProfileConfigured = errors.New("no mode profile configured")

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
	Advisories   []*errortypes.AdvisoryError
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
