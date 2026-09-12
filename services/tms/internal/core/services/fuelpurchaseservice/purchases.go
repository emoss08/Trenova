package fuelpurchaseservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/fuelimport"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) ListPurchases(
	ctx context.Context,
	req *repositories.ListFuelPurchasesRequest,
) (*pagination.CursorListResult[*fuelpurchase.FuelPurchase], error) {
	return s.repo.ListPurchases(ctx, req)
}

func (s *Service) GetPurchase(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*fuelpurchase.FuelPurchase, error) {
	return s.repo.GetPurchaseByID(ctx, &repositories.GetFuelPurchaseByIDRequest{
		ID:                  id,
		TenantInfo:          tenantInfo,
		IncludeTractor:      true,
		IncludeWorker:       true,
		IncludeJurisdiction: true,
		IncludeFuelCard:     true,
	})
}

type CreatePurchaseRequest struct {
	TenantInfo       pagination.TenantInfo
	Purchase         *fuelpurchase.FuelPurchase
	JurisdictionCode string
	UserID           pulid.ID
}

func (s *Service) CreatePurchase(
	ctx context.Context,
	req *CreatePurchaseRequest,
) (*fuelpurchase.FuelPurchase, error) {
	purchase := req.Purchase
	if purchase == nil {
		return nil, errortypes.NewValidationError(
			"purchase",
			errortypes.ErrRequired,
			"A purchase is required",
		)
	}

	purchase.ID = ""
	purchase.OrganizationID = req.TenantInfo.OrgID
	purchase.BusinessUnitID = req.TenantInfo.BuID
	purchase.Source = fuelpurchase.PurchaseSourceManual
	purchase.ImportBatchID = nil
	purchase.CreatedByID = req.UserID
	purchase.Normalize()

	err := s.resolvePurchaseReferences(ctx, req.TenantInfo, purchase, req.JurisdictionCode)
	if err != nil {
		return nil, err
	}
	if err = validateEntity(purchase); err != nil {
		return nil, err
	}

	created, err := s.repo.CreatePurchase(ctx, purchase)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceFuelPurchase,
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    created,
		comment: "Recorded " + created.Gallons.StringFixed(3) + " gallons of " +
			created.FuelType.Label(),
	})
	s.publish(ctx, req.TenantInfo, realtimePurchase, permission.OpCreate, created.ID, req.UserID)

	return created, nil
}

type UpdatePurchaseRequest struct {
	TenantInfo       pagination.TenantInfo
	Purchase         *fuelpurchase.FuelPurchase
	JurisdictionCode string
	UserID           pulid.ID
}

func (s *Service) UpdatePurchase(
	ctx context.Context,
	req *UpdatePurchaseRequest,
) (*fuelpurchase.FuelPurchase, error) {
	purchase := req.Purchase
	if purchase == nil {
		return nil, errortypes.NewValidationError(
			"purchase",
			errortypes.ErrRequired,
			"A purchase is required",
		)
	}

	stored, err := s.repo.GetPurchaseByID(ctx, &repositories.GetFuelPurchaseByIDRequest{
		ID:         purchase.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if purchase.Source != "" && purchase.Source != stored.Source {
		return nil, errortypes.NewValidationError(
			"source",
			errortypes.ErrInvalid,
			"The source of a purchase cannot be changed",
		)
	}
	if purchase.ImportBatchID != nil && !purchase.ImportBatchID.IsNil() &&
		!sameOptionalID(purchase.ImportBatchID, stored.ImportBatchID) {
		return nil, errortypes.NewValidationError(
			"importBatchId",
			errortypes.ErrInvalid,
			"A purchase cannot be moved between imports",
		)
	}

	purchase.OrganizationID = stored.OrganizationID
	purchase.BusinessUnitID = stored.BusinessUnitID
	purchase.Source = stored.Source
	purchase.ImportBatchID = stored.ImportBatchID
	purchase.CreatedByID = stored.CreatedByID
	purchase.CreatedAt = stored.CreatedAt
	purchase.Normalize()

	err = s.resolvePurchaseReferences(ctx, req.TenantInfo, purchase, req.JurisdictionCode)
	if err != nil {
		return nil, err
	}
	if err = validateEntity(purchase); err != nil {
		return nil, err
	}

	previous := *stored
	updated, err := s.repo.UpdatePurchase(ctx, purchase)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceFuelPurchase,
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   &previous,
		comment: "Updated fuel purchase; a finalized IFTA return covering this quarter " +
			"keeps its snapshot until it is reopened or amended",
	})
	s.publish(ctx, req.TenantInfo, realtimePurchase, permission.OpUpdate, updated.ID, req.UserID)

	return updated, nil
}

type DeletePurchaseRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Version    int64
	UserID     pulid.ID
}

func (s *Service) DeletePurchase(ctx context.Context, req *DeletePurchaseRequest) error {
	stored, err := s.repo.GetPurchaseByID(ctx, &repositories.GetFuelPurchaseByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}
	if stored.Version != req.Version {
		return dberror.CreateVersionMismatchError("FuelPurchase", stored.ID.String())
	}

	if err = s.repo.DeletePurchase(ctx, &repositories.DeleteFuelPurchaseRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceFuelPurchase,
		resourceID: stored.ID.String(),
		operation:  permission.OpDelete,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    stored,
		previous:   stored,
		comment: "Deleted fuel purchase; a finalized IFTA return covering this quarter " +
			"keeps its snapshot until it is reopened or amended",
	})
	s.publish(ctx, req.TenantInfo, realtimePurchase, permission.OpDelete, stored.ID, req.UserID)

	return nil
}

func (s *Service) resolvePurchaseReferences(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	purchase *fuelpurchase.FuelPurchase,
	jurisdictionCode string,
) error {
	if purchase.TractorID.IsNil() {
		return errortypes.NewValidationError(
			"tractorId",
			errortypes.ErrRequired,
			"Tractor is required",
		)
	}
	if _, err := s.tractorRepo.GetByID(ctx, repositories.GetTractorByIDRequest{
		ID:         purchase.TractorID,
		TenantInfo: tenantInfo,
	}); err != nil {
		return referenceError(err, "tractorId", "Tractor does not exist in your organization")
	}

	if purchase.WorkerID != nil && !purchase.WorkerID.IsNil() {
		if _, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
			ID:         *purchase.WorkerID,
			TenantInfo: tenantInfo,
		}); err != nil {
			return referenceError(err, "workerId", "Worker does not exist in your organization")
		}
	} else {
		purchase.WorkerID = nil
	}

	jurisdiction, err := s.resolveJurisdiction(ctx, purchase.JurisdictionID, jurisdictionCode)
	if err != nil {
		return err
	}
	purchase.JurisdictionID = jurisdiction.ID

	return s.resolvePurchaseCard(ctx, tenantInfo, purchase)
}

func (s *Service) resolveJurisdiction(
	ctx context.Context,
	id pulid.ID,
	code string,
) (*ifta.Jurisdiction, error) {
	var (
		jurisdiction *ifta.Jurisdiction
		err          error
	)

	switch {
	case !id.IsNil():
		jurisdiction, err = s.jurisdictions.GetJurisdictionByID(ctx, id)
	case strings.TrimSpace(code) != "":
		country, jurisdictionCode, ok := fuelimport.NormalizeJurisdiction(code)
		if !ok {
			return nil, errortypes.NewValidationError(
				"jurisdictionId",
				errortypes.ErrInvalid,
				"\"{0}\" is not a recognised state or province", strings.TrimSpace(code),
			)
		}
		jurisdiction, err = s.jurisdictions.GetJurisdictionByCode(ctx, country, jurisdictionCode)
	default:
		return nil, errortypes.NewValidationError(
			"jurisdictionId",
			errortypes.ErrRequired,
			"Jurisdiction is required",
		)
	}
	if err != nil {
		return nil, referenceError(err, "jurisdictionId", "Jurisdiction does not exist")
	}
	if jurisdiction == nil {
		return nil, errortypes.NewValidationError(
			"jurisdictionId",
			errortypes.ErrInvalid,
			"Jurisdiction does not exist",
		)
	}
	if !jurisdiction.IsActive() {
		return nil, errortypes.NewValidationError(
			"jurisdictionId",
			errortypes.ErrInvalid,
			"{0} is not an active jurisdiction", jurisdiction.Label(),
		)
	}

	return jurisdiction, nil
}

func (s *Service) resolvePurchaseCard(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	purchase *fuelpurchase.FuelPurchase,
) error {
	if purchase.FuelCardID != nil && !purchase.FuelCardID.IsNil() {
		card, err := s.repo.GetCardByID(ctx, &repositories.GetFuelCardByIDRequest{
			ID:         *purchase.FuelCardID,
			TenantInfo: tenantInfo,
		})
		if err != nil {
			return referenceError(
				err,
				"fuelCardId",
				"Fuel card does not exist in your organization",
			)
		}
		if purchase.CardLastFour == "" {
			purchase.CardLastFour = card.LastFour
		}
		if purchase.WorkerID == nil && card.AssignedWorkerID != nil {
			purchase.WorkerID = card.AssignedWorkerID
		}
		return nil
	}

	purchase.FuelCardID = nil
	if purchase.CardLastFour == "" {
		return nil
	}

	card, err := s.repo.FindCardByLastFour(ctx, &repositories.FindFuelCardByLastFourRequest{
		TenantInfo: tenantInfo,
		LastFour:   purchase.CardLastFour,
	})
	if err != nil {
		return err
	}
	if card != nil {
		purchase.FuelCardID = &card.ID
		if purchase.WorkerID == nil && card.AssignedWorkerID != nil {
			purchase.WorkerID = card.AssignedWorkerID
		}
	}

	return nil
}

func referenceError(err error, field, message string) error {
	if errortypes.IsNotFoundError(err) {
		return errortypes.NewValidationError(field, errortypes.ErrInvalid, message)
	}
	return err
}

func sameOptionalID(a, b *pulid.ID) bool {
	aNil := a == nil || a.IsNil()
	bNil := b == nil || b.IsNil()
	if aNil || bNil {
		return aNil == bNil
	}
	return *a == *b
}
