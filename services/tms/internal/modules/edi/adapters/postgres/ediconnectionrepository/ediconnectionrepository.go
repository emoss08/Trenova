//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package ediconnectionrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	connectionErrors     = dberror.Mapping{Entity: "EDIConnection"}
	partnerErrors        = dberror.Mapping{Entity: "EDIPartner"}
	profileErrors        = dberror.Mapping{Entity: "EDICommunicationProfile"}
	mappingProfileErrors = dberror.Mapping{Entity: "EDIMappingProfile"}
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) edipartnerports.EDIConnectionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-connection-repository"),
	}
}

func (r *repository) ListConnections(
	ctx context.Context,
	req *edipartnerports.ListEDIConnectionsRequest,
) (*pagination.ListResult[*edi.EDIConnection], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDIConnection], error) {
			ec := edidb.EDIConnections
			q := ec.Where(involvesOrganization(req.Filter.TenantInfo.OrgID))
			if req.Filter.Query != "" {
				q.Where(keel.Or(
					listquery.TextContainsFold(ec.Method, req.Filter.Query),
					listquery.TextContainsFold(ec.Status, req.Filter.Query),
					ec.SourceOrganization().Name.ContainsFold(req.Filter.Query),
					ec.TargetOrganization().Name.ContainsFold(req.Filter.Query),
				))
			}
			q.Load(
				ec.SourceOrganization(),
				ec.TargetOrganization(),
				ec.SourcePartner(),
				ec.TargetPartner(),
			).Order(ec.CreatedAt.Desc())

			return listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
		},
	)
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *edipartnerports.EDIConnectionSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIConnection], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDIConnection], error) {
			ec := edidb.EDIConnections
			q := ec.Pick(
				ec.ID,
				ec.Method,
				ec.SourceOrganizationID,
				ec.TargetOrganizationID,
				ec.Status,
				ec.SourcePartnerID,
				ec.TargetPartnerID,
				ec.Version,
				ec.CreatedAt,
				ec.UpdatedAt,
			).
				Where(
					ec.SourceOrganizationID.Eq(req.SelectQueryRequest.TenantInfo.OrgID),
					ec.Status.Eq(edi.ConnectionStatusActive),
				).
				Load(ec.SourceOrganization(), ec.TargetOrganization()).
				Order(ec.CreatedAt.Desc())

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(
						listquery.TextContainsFold(ec.Method, s),
						ec.SourceOrganization().Name.ContainsFold(s),
						ec.TargetOrganization().Name.ContainsFold(s),
					)
				},
			)
		},
	)
}

func (r *repository) GetConnectionsByIDs(
	ctx context.Context,
	req edipartnerports.GetEDIConnectionsByIDsRequest,
) ([]*edi.EDIConnection, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.EDIConnection, error) {
		if len(req.ConnectionIDs) == 0 {
			return []*edi.EDIConnection{}, nil
		}

		ec := edidb.EDIConnections
		return ec.Where(
			ec.ID.In(req.ConnectionIDs...),
			involvesOrganization(req.TenantInfo.OrgID),
		).
			Load(ec.SourceOrganization(), ec.TargetOrganization()).
			Limit(len(req.ConnectionIDs)).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) GetConnectionByID(
	ctx context.Context,
	req edipartnerports.GetEDIConnectionByIDRequest,
) (*edi.EDIConnection, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDIConnection, error) {
		ec := edidb.EDIConnections
		entity, err := ec.Where(ec.ID.Eq(req.ID), involvesOrganization(req.TenantInfo.OrgID)).
			Load(
				ec.SourceOrganization(),
				ec.TargetOrganization(),
				ec.SourcePartner(),
				ec.TargetPartner(),
			).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, connectionErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetConnectionForUpdate(
	ctx context.Context,
	req edipartnerports.GetEDIConnectionForUpdateRequest,
) (*edi.EDIConnection, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDIConnection, error) {
		ec := edidb.EDIConnections
		entity, err := ec.Where(ec.ID.Eq(req.ID), involvesOrganization(req.TenantInfo.OrgID)).
			Lock(keel.ForUpdate).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, connectionErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetActiveConnectionForPartner(
	ctx context.Context,
	req edipartnerports.GetActiveEDIConnectionForPartnerRequest,
) (*edi.EDIConnection, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDIConnection, error) {
		ec := edidb.EDIConnections
		org := req.TenantInfo.OrgID
		entity, err := ec.Where(
			ec.Method.Eq(req.Method),
			ec.Status.Eq(edi.ConnectionStatusActive),
			keel.Or(
				keel.And(ec.SourceOrganizationID.Eq(org), ec.SourcePartnerID.Eq(req.PartnerID)),
				keel.And(ec.TargetOrganizationID.Eq(org), ec.TargetPartnerID.Eq(req.PartnerID)),
			),
		).First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, connectionErrors.Map(err, req.PartnerID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateConnection(
	ctx context.Context,
	entity *edi.EDIConnection,
) (*edi.EDIConnection, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDIConnection, error) {
		if err := r.ensureTargetOrganizationInBusinessUnit(
			ctx,
			entity.TargetOrganizationID,
			entity.BusinessUnitID,
		); err != nil {
			return nil, err
		}

		if err := edidb.EDIConnections.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, connectionErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateConnection(
	ctx context.Context,
	entity *edi.EDIConnection,
) (*edi.EDIConnection, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDIConnection, error) {
		if err := edidb.EDIConnections.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, connectionErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) AcceptInternalConnection(
	ctx context.Context,
	req *edipartnerports.CreateInternalEDIConnectionAcceptanceRequest,
) (*edi.EDIConnection, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"create the partner and profile records in both organizations of an internal connection atomically",
	)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDIConnection, error) {
		db := r.db.Keel(ctx)
		ep := edidb.EDIPartners
		cp := edidb.EDICommunicationProfiles

		for _, partner := range []*edi.EDIPartner{req.SourcePartner, req.TargetPartner} {
			if err := ep.Create(ctx, db, partner); err != nil {
				return nil, partnerErrors.Map(err, partner.ID.String())
			}
			if err := ensureMappingProfile(ctx, db, partner); err != nil {
				return nil, err
			}
		}

		req.SourceProfile.EDIPartnerID = req.SourcePartner.ID
		req.TargetProfile.EDIPartnerID = req.TargetPartner.ID
		for _, profile := range []*edi.EDICommunicationProfile{req.SourceProfile, req.TargetProfile} {
			if err := cp.Create(ctx, db, profile); err != nil {
				return nil, profileErrors.Map(err, profile.ID.String())
			}
		}

		req.SourcePartner.DefaultTransportID = req.SourceProfile.ID
		req.TargetPartner.DefaultTransportID = req.TargetProfile.ID
		for _, partner := range []*edi.EDIPartner{req.SourcePartner, req.TargetPartner} {
			if err := ep.Save(ctx, db, partner, ep.DefaultTransportID); err != nil {
				return nil, partnerErrors.Map(err, partner.ID.String())
			}
		}

		req.Connection.SourcePartnerID = req.SourcePartner.ID
		req.Connection.TargetPartnerID = req.TargetPartner.ID
		if err := edidb.EDIConnections.Save(ctx, db, req.Connection); err != nil {
			return nil, connectionErrors.Map(err, req.Connection.ID.String())
		}

		req.Connection.SourcePartner = req.SourcePartner
		req.Connection.TargetPartner = req.TargetPartner
		return req.Connection, nil
	})
}

func ensureMappingProfile(ctx context.Context, db *keel.DB, partner *edi.EDIPartner) error {
	mp := edidb.EDIMappingProfiles
	exists, err := mp.AllTenants().Where(
		mp.OrganizationID.Eq(partner.OrganizationID),
		mp.BusinessUnitID.Eq(partner.BusinessUnitID),
		mp.EDIPartnerID.Eq(partner.ID),
	).Exists(ctx, db)
	if err != nil || exists {
		return err
	}

	profile := &edi.EDIMappingProfile{
		BusinessUnitID: partner.BusinessUnitID,
		OrganizationID: partner.OrganizationID,
		EDIPartnerID:   partner.ID,
		Name:           partner.Name + " Mapping Profile",
	}
	if err = mp.Create(ctx, db, profile); err != nil {
		return mappingProfileErrors.Map(err, profile.ID.String())
	}

	return nil
}

func (r *repository) ensureTargetOrganizationInBusinessUnit(
	ctx context.Context,
	targetOrganizationID pulid.ID,
	businessUnitID pulid.ID,
) error {
	ctx = dbscope.WithSystem(
		ctx,
		"confirm that an organization not yet connected belongs to the caller's business unit",
	)
	return dbtx.ReadErr(ctx, r.db, func(ctx context.Context) error {
		org := tenantdb.Organizations
		exists, err := org.AllTenants().Where(
			org.ID.Eq(targetOrganizationID),
			org.BusinessUnitID.Eq(businessUnitID),
		).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			return err
		}
		if exists {
			return nil
		}

		return errortypes.NewValidationError(
			"targetOrganizationId",
			errortypes.ErrInvalid,
			"Target organization must belong to the current business unit",
		)
	})
}

func involvesOrganization(orgID pulid.ID) keel.Pred {
	ec := edidb.EDIConnections
	return keel.Or(ec.SourceOrganizationID.Eq(orgID), ec.TargetOrganizationID.Eq(orgID))
}
