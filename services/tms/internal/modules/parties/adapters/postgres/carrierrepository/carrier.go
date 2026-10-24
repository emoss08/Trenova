package carrierrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var carrierErrors = dberror.Mapping{
	Entity: "Carrier",
	Unique: []dberror.Constraint{
		{
			Name:    "idx_carriers_code",
			Field:   "code",
			Message: "Carrier with this code already exists in your organization",
		},
		{
			Name:    "idx_carriers_dot_number",
			Field:   "dotNumber",
			Message: "Carrier with this DOT number already exists in your organization",
		},
		{
			Name:    "idx_carriers_scac",
			Field:   "scac",
			Message: "Carrier with this SCAC already exists in your organization",
		},
		{
			Name:    "uq_carrier_edi_channels_default",
			Field:   "ediChannels",
			Message: "Carrier can have only one default EDI channel",
		},
		{
			Name:    "uq_carrier_edi_channels_partner",
			Field:   "ediChannels",
			Message: "Carrier already has an EDI channel for this partner",
		},
	},
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) carrierports.CarrierRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-repository"),
	}
}

func carrierLoads(opts carrierports.CarrierFilterOptions) []keel.Loader {
	c := carrierdb.Carriers
	ch := c.EDIChannels()

	loads := make([]keel.Loader, 0, 5)
	if opts.IncludeState {
		loads = append(loads, c.State(), c.RemitState())
	}
	if opts.IncludeContacts {
		loads = append(loads, c.Contacts())
	}
	if opts.IncludeInsurancePolicies {
		loads = append(loads, c.InsurancePolicies())
	}
	if opts.IncludeEDIChannels {
		loads = append(loads, ch.Load(ch.EDIPartner()))
	}

	return loads
}

func (r *repository) List(
	ctx context.Context,
	req *carrierports.ListCarrierRequest,
) (*pagination.ListResult[*carrier.Carrier], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*carrier.Carrier], error) {
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				carrierdb.Carriers.Query(),
				&listquery.Request[carrier.Carrier]{
					Filter: req.Filter,
					Shape: func(q *keel.Query[carrier.Carrier]) *keel.Query[carrier.Carrier] {
						return q.Load(carrierLoads(req.CarrierFilterOptions)...)
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list carriers", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *carrierports.ListCarrierConnectionRequest,
) (*pagination.CursorListResult[*carrier.Carrier], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*carrier.Carrier], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				carrierdb.Carriers.Query(),
				&listquery.Request[carrier.Carrier]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.CarrierColumns,
					Shape: func(q *keel.Query[carrier.Carrier]) *keel.Query[carrier.Carrier] {
						return q.Load(carrierLoads(req.CarrierFilterOptions)...)
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list carrier connection", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req carrierports.GetCarrierByIDRequest,
) (*carrier.Carrier, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*carrier.Carrier, error) {
		c := carrierdb.Carriers
		entity, err := c.Where(c.ID.Eq(req.ID)).
			Load(carrierLoads(req.CarrierFilterOptions)...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get carrier", zap.String("id", req.ID.String()), zap.Error(err))
			return nil, carrierErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) FindByIdentity(
	ctx context.Context,
	req *carrierports.FindCarrierByIdentityRequest,
) (*carrier.Carrier, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*carrier.Carrier, error) {
		if req.SCAC == "" && req.DOTNumber == "" {
			return nil, nil //nolint:nilnil // nothing to look up without an identity
		}

		c := carrierdb.Carriers
		entity, err := c.Where(keel.Or(
			keel.When(req.SCAC != "", keel.Raw("upper(?) = ?", c.SCAC, strings.ToUpper(req.SCAC))),
			keel.When(req.DOTNumber != "", c.DOTNumber.Eq(req.DOTNumber)),
		)).
			Order(c.CreatedAt.Asc()).
			First(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nil carrier represents an optional absence
		}
		if err != nil {
			return nil, fmt.Errorf("find carrier by identity: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req carrierports.GetCarriersByIDsRequest,
) ([]*carrier.Carrier, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*carrier.Carrier, error) {
		if len(req.CarrierIDs) == 0 {
			return []*carrier.Carrier{}, nil
		}

		c := carrierdb.Carriers
		entities, err := c.Where(c.ID.In(req.CarrierIDs...)).
			Load(carrierLoads(req.CarrierFilterOptions)...).
			Limit(len(req.CarrierIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get carriers", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *carrierports.CarrierSelectOptionsRequest,
) (*pagination.ListResult[*carrier.Carrier], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*carrier.Carrier], error) {
			c := carrierdb.Carriers
			q := c.Pick(c.ID, c.Code, c.Name).Where(c.Status.Eq(carrier.StatusActive))

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(c.Code.ContainsFold(s), c.Name.ContainsFold(s))
				},
			)
		},
	)
}

// stampChildren pins every child row to its parent and tenant. IDs are left
// alone: rows the caller kept retain their identity (and created_at history);
// keel mints IDs for the new, ID-less rows only.
func (r *repository) stampChildren(entity *carrier.Carrier) {
	for _, contact := range entity.Contacts {
		contact.CarrierID = entity.ID
		contact.OrganizationID = entity.OrganizationID
		contact.BusinessUnitID = entity.BusinessUnitID
	}

	for _, policy := range entity.InsurancePolicies {
		policy.CarrierID = entity.ID
		policy.OrganizationID = entity.OrganizationID
		policy.BusinessUnitID = entity.BusinessUnitID
	}

	for _, channel := range entity.EDIChannels {
		channel.CarrierID = entity.ID
		channel.OrganizationID = entity.OrganizationID
		channel.BusinessUnitID = entity.BusinessUnitID
	}
}

func (r *repository) insertChildren(ctx context.Context, entity *carrier.Carrier) error {
	db := r.db.Keel(ctx)
	if err := carrierdb.CarrierContacts.CreateMany(ctx, db, entity.Contacts); err != nil {
		return err
	}
	if err := carrierdb.CarrierInsurancePolicies.CreateMany(
		ctx,
		db,
		entity.InsurancePolicies,
	); err != nil {
		return err
	}

	return carrierdb.CarrierEDIChannels.CreateMany(ctx, db, entity.EDIChannels)
}

// deleteMissingChildren removes only the child rows the payload no longer
// carries; rows whose IDs are still present keep their identity and history.
func (r *repository) deleteMissingChildren(ctx context.Context, entity *carrier.Carrier) error {
	db := r.db.Keel(ctx)

	cc := carrierdb.CarrierContacts
	kept := pulid.Keep(entity.Contacts)
	if _, err := cc.Where(
		cc.CarrierID.Eq(entity.ID),
		keel.When(len(kept) > 0, cc.ID.NotIn(kept...)),
	).Delete(ctx, db); err != nil {
		return err
	}

	cip := carrierdb.CarrierInsurancePolicies
	kept = pulid.Keep(entity.InsurancePolicies)
	if _, err := cip.Where(
		cip.CarrierID.Eq(entity.ID),
		keel.When(len(kept) > 0, cip.ID.NotIn(kept...)),
	).Delete(ctx, db); err != nil {
		return err
	}

	cec := carrierdb.CarrierEDIChannels
	kept = pulid.Keep(entity.EDIChannels)
	_, err := cec.Where(
		cec.CarrierID.Eq(entity.ID),
		keel.When(len(kept) > 0, cec.ID.NotIn(kept...)),
	).Delete(ctx, db)

	return err
}

// clearEDIChannelDefaults drops every default flag before the channel upsert:
// uq_carrier_edi_channels_default is a non-deferrable partial unique index, so
// moving is_default between two existing rows in one statement would collide
// mid-upsert.
func (r *repository) clearEDIChannelDefaults(ctx context.Context, entity *carrier.Carrier) error {
	cec := carrierdb.CarrierEDIChannels
	_, err := cec.Where(cec.CarrierID.Eq(entity.ID), cec.IsDefault.IsTrue()).Update(
		ctx,
		r.db.Keel(ctx),
		cec.IsDefault.To(false),
		cec.Version.ToExpr(cec.Version),
		cec.UpdatedAt.ToExpr(cec.UpdatedAt),
	)

	return err
}

// upsertChildren inserts new (ID-less) child rows and updates the kept ones in
// place, preserving their IDs and created_at for insurance history and audit
// diffs.
func (r *repository) upsertChildren(ctx context.Context, entity *carrier.Carrier) error {
	db := r.db.Keel(ctx)

	if len(entity.Contacts) > 0 {
		cc := carrierdb.CarrierContacts
		inserted, err := cc.Upsert(ctx, db, entity.Contacts,
			keel.OnConflict(cc.ID, cc.BusinessUnitID, cc.OrganizationID).Update(
				cc.Name,
				cc.Title,
				cc.Email,
				cc.Phone,
				cc.IsPrimary,
				cc.ReceivesRateConfirmations,
			))
		if err != nil {
			return err
		}
		if err = restoreStored(entity.Contacts, inserted, func(ids []pulid.ID) ([]*carrier.CarrierContact, error) {
			return cc.Where(cc.ID.In(ids...)).All(ctx, db)
		}); err != nil {
			return err
		}
	}

	if len(entity.InsurancePolicies) > 0 {
		cip := carrierdb.CarrierInsurancePolicies
		inserted, err := cip.Upsert(ctx, db, entity.InsurancePolicies,
			keel.OnConflict(cip.ID, cip.BusinessUnitID, cip.OrganizationID).Update(
				cip.PolicyType,
				cip.PolicyNumber,
				cip.ProviderName,
				cip.CoverageAmount,
				cip.EffectiveDate,
				cip.ExpirationDate,
				cip.IsVerified,
			))
		if err != nil {
			return err
		}
		if err = restoreStored(entity.InsurancePolicies, inserted, func(ids []pulid.ID) ([]*carrier.CarrierInsurancePolicy, error) {
			return cip.Where(cip.ID.In(ids...)).All(ctx, db)
		}); err != nil {
			return err
		}
	}

	if len(entity.EDIChannels) > 0 {
		if err := r.clearEDIChannelDefaults(ctx, entity); err != nil {
			return err
		}

		cec := carrierdb.CarrierEDIChannels
		inserted, err := cec.Upsert(ctx, db, entity.EDIChannels,
			keel.OnConflict(cec.ID, cec.BusinessUnitID, cec.OrganizationID).Update(
				cec.EDIPartnerID,
				cec.PartnerDocumentProfileID,
				cec.CommunicationProfileID,
				cec.SCACOverride,
				cec.IsDefault,
			))
		if err != nil {
			return err
		}
		if err = restoreStored(entity.EDIChannels, inserted, func(ids []pulid.ID) ([]*carrier.CarrierEDIChannel, error) {
			return cec.Where(cec.ID.In(ids...)).All(ctx, db)
		}); err != nil {
			return err
		}
	}

	return nil
}

func restoreStored[T interface{ GetID() pulid.ID }](
	rows []T,
	inserted []bool,
	read func(ids []pulid.ID) ([]T, error),
) error {
	ids := make([]pulid.ID, 0, len(rows))
	for i, wasInserted := range inserted {
		if !wasInserted {
			ids = append(ids, rows[i].GetID())
		}
	}
	if len(ids) == 0 {
		return nil
	}

	stored, err := read(ids)
	if err != nil {
		return err
	}

	byID := make(map[pulid.ID]T, len(stored))
	for _, row := range stored {
		byID[row.GetID()] = row
	}
	for i, wasInserted := range inserted {
		if row, ok := byID[rows[i].GetID()]; ok && !wasInserted {
			rows[i] = row
		}
	}

	return nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *carrier.Carrier,
) (*carrier.Carrier, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*carrier.Carrier, error) {
		err := carrierdb.Carriers.Create(ctx, r.db.Keel(ctx), entity)
		if err == nil {
			r.stampChildren(entity)
			err = r.insertChildren(ctx, entity)
		}
		if err != nil {
			r.l.Error("failed to create carrier", zap.String("code", entity.Code), zap.Error(err))
			return nil, carrierErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *carrier.Carrier,
) (*carrier.Carrier, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*carrier.Carrier, error) {
		c := carrierdb.Carriers
		err := c.SaveExcept(
			ctx,
			r.db.Keel(ctx),
			entity,
			c.IntelRiskLevel,
			c.IntelReviewRequired,
			c.IntelBlockingCount,
		)
		if err == nil {
			err = r.deleteMissingChildren(ctx, entity)
		}
		if err == nil {
			r.stampChildren(entity)
			err = r.upsertChildren(ctx, entity)
		}
		if err != nil {
			r.l.Error(
				"failed to update carrier",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, dberror.MapRetryableTransactionError(
				carrierErrors.Map(err, entity.ID.String()),
				"Carrier is busy. Retry the request.",
			)
		}

		return entity, nil
	})
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *carrierports.BulkUpdateCarrierStatusRequest,
) ([]*carrier.Carrier, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*carrier.Carrier, error) {
		c := carrierdb.Carriers
		entities, err := listquery.UpdateReturning(
			ctx,
			r.db.Keel(ctx),
			c.Where(c.ID.In(req.CarrierIDs...)),
			c.Status.To(req.Status),
		)
		if err != nil {
			r.l.Error("failed to bulk update carrier status", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckBulkAffected(
			int64(len(entities)),
			"Carrier",
			req.CarrierIDs,
		); err != nil {
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) ListEDIChannels(
	ctx context.Context,
	req carrierports.GetCarrierEDIChannelRequest,
) ([]*carrier.CarrierEDIChannel, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*carrier.CarrierEDIChannel, error) {
		cec := carrierdb.CarrierEDIChannels
		entities, err := cec.Where(cec.CarrierID.Eq(req.CarrierID)).
			Load(cec.EDIPartner()).
			Order(cec.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list carrier EDI channels", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) ListEDIChannelsByPartner(
	ctx context.Context,
	req carrierports.ListCarrierEDIChannelsByPartnerRequest,
) ([]*carrier.CarrierEDIChannel, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*carrier.CarrierEDIChannel, error) {
		cec := carrierdb.CarrierEDIChannels
		entities, err := cec.Where(cec.EDIPartnerID.Eq(req.EDIPartnerID)).
			Load(cec.Carrier()).
			Order(cec.IsDefault.Desc(), cec.CreatedAt.Asc(), cec.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list carrier EDI channels by partner", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) GetDefaultEDIChannel(
	ctx context.Context,
	req carrierports.GetCarrierEDIChannelRequest,
) (*carrier.CarrierEDIChannel, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*carrier.CarrierEDIChannel, error) {
		cec := carrierdb.CarrierEDIChannels
		entity, err := cec.Where(cec.CarrierID.Eq(req.CarrierID), cec.IsDefault.IsTrue()).
			Load(cec.EDIPartner()).
			One(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nil channel represents an optional absence
		}
		if err != nil {
			r.l.Error("failed to get default carrier EDI channel", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}
