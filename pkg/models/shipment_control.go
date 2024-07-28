package models

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/audit"
	"github.com/emoss08/trenova/pkg/constants"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/validator"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ShipmentControl struct {
	bun.BaseModel `bun:"table:shipment_controls,alias:sc" json:"-"`
	ID            uuid.UUID `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`

	EnforceRevCode           bool      `bun:"type:BOOLEAN,notnull,default:false" json:"enforceRevCode"`
	EnforceVoidedComm        bool      `bun:"type:BOOLEAN,notnull,default:false" json:"enforceVoidedComm"`
	AutoTotalShipment        bool      `bun:"type:BOOLEAN,notnull,default:false" json:"autoTotalShipment"`
	CompareOriginDestination bool      `bun:"type:BOOLEAN,notnull,default:false" json:"compareOriginDestination"`
	CheckForDuplicateBol     bool      `bun:"type:BOOLEAN,notnull,default:false" json:"checkForDuplicateBol"`
	Version                  int64     `bun:"type:BIGINT" json:"version"`
	CreatedAt                time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt                time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
}

func QueryShipmentControlByOrgID(ctx context.Context, tx bun.IDB, orgID uuid.UUID) (*ShipmentControl, error) {
	var shipmentControl ShipmentControl
	err := tx.NewSelect().Model(&shipmentControl).Where("sc.organization_id = ?", orgID).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &shipmentControl, nil
}

func (sc *ShipmentControl) BeforeUpdate(_ context.Context) error {
	sc.Version++

	return nil
}

func (c *ShipmentControl) Insert(ctx context.Context, tx bun.IDB, auditService *audit.Service, user audit.AuditUser) error {
	//if err := c.Validate(); err != nil {
	//	return err
	//}

	if _, err := tx.NewInsert().Model(c).Returning("*").Exec(ctx); err != nil {
		return err
	}

	auditService.LogAction(
		constants.TableShipmentControl,
		c.ID.String(),
		property.AuditLogActionCreate,
		user,
		c.OrganizationID,
		c.BusinessUnitID,
		audit.WithDiff(nil, c),
	)

	return nil
}

func (c *ShipmentControl) UpdateOne(ctx context.Context, tx bun.IDB, auditService *audit.Service, user audit.AuditUser) error {
	original := new(ShipmentControl)
	if err := tx.NewSelect().Model(original).Where("id = ?", c.ID).Scan(ctx); err != nil {
		return validator.BusinessLogicError{Message: err.Error()}
	}

	//if err := c.Validate(); err != nil {
	//	return err
	//}

	if err := c.OptimisticUpdate(ctx, tx); err != nil {
		return err
	}

	auditService.LogAction(
		constants.TableShipmentControl,
		c.ID.String(),
		property.AuditLogActionUpdate,
		user,
		c.OrganizationID,
		c.BusinessUnitID,
		audit.WithDiff(original, c),
	)

	return nil
}

func (sc *ShipmentControl) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
	ov := sc.Version

	if err := sc.BeforeUpdate(ctx); err != nil {
		return err
	}

	result, err := tx.NewUpdate().
		Model(sc).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return &validator.BusinessLogicError{
			Message: fmt.Sprintf("Version mismatch. The Shipment Control (ID: %s) has been updated by another user. Please refresh and try again.", sc.ID),
		}
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*ShipmentControl)(nil)

func (sc *ShipmentControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		sc.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		sc.UpdatedAt = time.Now()
	}

	return nil
}
