package edipartnerrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/ediscope"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	partnerErrors           = dberror.Mapping{Entity: "EDIPartner"}
	reciprocalPartnerErrors = dberror.Mapping{Entity: "ReciprocalEDIPartner"}
	mappingProfileErrors    = dberror.Mapping{Entity: "EDIMappingProfile"}
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

func New(p Params) ediports.EDIPartnerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-partner-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *ediports.ListEDIPartnersRequest,
) (*pagination.ListResult[*edi.EDIPartner], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDIPartner], error) {
			ep := edidb.EDIPartners
			return listquery.List(
				ctx,
				r.db.Keel(ctx),
				listedPartners(req),
				&listquery.Request[edi.EDIPartner]{
					Filter: req.Filter,
					Shape:  withLinks,
					Then:   []keel.Order{ep.CreatedAt.Desc()},
				},
			)
		},
	)
}

func (r *repository) ListCursor(
	ctx context.Context,
	req *ediports.ListEDIPartnersRequest,
) (*pagination.CursorListResult[*edi.EDIPartner], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*edi.EDIPartner], error) {
			return listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				listedPartners(req),
				&listquery.Request[edi.EDIPartner]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape:  withLinks,
				},
			)
		},
	)
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *ediports.EDIPartnerSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIPartner], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDIPartner], error) {
			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				partnerOptionsQuery(req),
				req.SelectQueryRequest,
				partnerSearch,
			)
		},
	)
}

func partnerOptionsQuery(req *ediports.EDIPartnerSelectOptionsRequest) *keel.Query[edi.EDIPartner] {
	ep := edidb.EDIPartners
	return ep.Pick(
		ep.ID,
		ep.BusinessUnitID,
		ep.OrganizationID,
		ep.Kind,
		ep.Code,
		ep.Name,
		ep.InternalOrganizationID,
		ep.EDIConnectionID,
		ep.DefaultTransportID,
		ep.EnabledForInbound,
		ep.EnabledForOutbound,
	).Where(
		keel.When(req.Kind != "", ep.Kind.Eq(req.Kind)),
		keel.When(req.EnabledForOutbound, ep.EnabledForOutbound.IsTrue()),
	)
}

func partnerSearch(s string) keel.Pred {
	ep := edidb.EDIPartners
	return keel.Or(ep.Name.ContainsFold(s), ep.Code.ContainsFold(s))
}

func (r *repository) GetByID(
	ctx context.Context,
	req ediports.GetEDIPartnerByIDRequest,
) (*edi.EDIPartner, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDIPartner, error) {
		ep := edidb.EDIPartners
		q := ediscope.Tenant(ctx, ep.Where(ep.ID.Eq(req.ID)), req.TenantInfo)
		if req.IncludeLinks {
			q.Load(ep.Customer(), ep.InternalOrganization(), ep.Connection())
		}

		entity, err := q.One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, partnerErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Create(ctx context.Context, entity *edi.EDIPartner) (*edi.EDIPartner, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDIPartner, error) {
		db := r.db.Keel(ctx)
		if err := edidb.EDIPartners.Create(ctx, db, entity); err != nil {
			return nil, partnerErrors.Map(err, entity.ID.String())
		}
		if err := ensureMappingProfile(ctx, db, entity); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func ensureMappingProfile(ctx context.Context, db *keel.DB, partner *edi.EDIPartner) error {
	mp := edidb.EDIMappingProfiles
	exists, err := mp.Where(mp.EDIPartnerID.Eq(partner.ID)).Exists(ctx, db)
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

func (r *repository) GetReadiness(
	ctx context.Context,
	req *ediports.GetEDIPartnerReadinessRequest,
) ([]*ediports.EDIPartnerReadinessRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*ediports.EDIPartnerReadinessRow, error) {
			if len(req.PartnerIDs) == 0 {
				return []*ediports.EDIPartnerReadinessRow{}, nil
			}

			rows, err := keel.Into[ediports.EDIPartnerReadinessRow](
				ctx,
				r.db.Keel(ctx),
				readinessQuery(req),
			)
			if err != nil {
				return nil, err
			}

			out := make([]*ediports.EDIPartnerReadinessRow, len(rows))
			for i := range rows {
				out[i] = &rows[i]
			}

			return out, nil
		},
	)
}

func readinessQuery(req *ediports.GetEDIPartnerReadinessRequest) *keel.Query[edi.EDIPartner] {
	ep := edidb.EDIPartners
	cp := edidb.EDICommunicationProfiles
	mi := edidb.EDIMappingProfileItems
	dp := edidb.EDIPartnerDocumentProfiles
	tc := edidb.EDITestCases

	activeDocumentProfile := func(direction edi.DocumentDirection) keel.Pred {
		return keel.Exists(dp.Where(
			dp.EDIPartnerID.EqCol(ep.ID),
			dp.Status.Eq(edi.DocumentStatusActive),
			dp.Direction.Eq(direction),
		))
	}

	return ep.Pick(
		ep.ID,
		ep.ContactEmail.Coalesce(""),
		ep.Timezone.Coalesce(""),
		keel.Exists(cp.Where(cp.EDIPartnerID.EqCol(ep.ID), cp.Status.Eq(domaintypes.StatusActive))),
		keel.Exists(mi.Where(mi.EDIPartnerID.EqCol(ep.ID))),
		activeDocumentProfile(edi.DocumentDirectionInbound),
		activeDocumentProfile(edi.DocumentDirectionOutbound),
		keel.Exists(tc.Where(
			tc.DocumentProfile().EDIPartnerID.EqCol(ep.ID),
			tc.LastRunPassed.IsTrue(),
		)),
		ep.EnabledForInbound,
		ep.EnabledForOutbound,
		keel.Raw("?::text", ep.Kind),
	).Where(ep.ID.In(req.PartnerIDs...))
}

func (r *repository) Update(ctx context.Context, entity *edi.EDIPartner) (*edi.EDIPartner, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDIPartner, error) {
		ep := edidb.EDIPartners
		err := ep.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			ep.Kind,
			ep.Status,
			ep.Code,
			ep.Name,
			ep.Description,
			ep.InternalOrganizationID,
			ep.EDIConnectionID,
			ep.CustomerID,
			ep.DefaultTransportID,
			ep.DefaultMappingProfileID,
			ep.Timezone,
			ep.Country,
			ep.ContactName,
			ep.ContactEmail,
			ep.ContactPhone,
			ep.EnabledForInbound,
			ep.EnabledForOutbound,
			ep.Settings,
		)
		if err != nil {
			return nil, partnerErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetReciprocalInternalPartner(
	ctx context.Context,
	req ediports.GetReciprocalInternalPartnerRequest,
) (*edi.EDIPartner, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"read the receiving organization's internal partner record for an internal tender",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDIPartner, error) {
		ep := edidb.EDIPartners
		org := tenantdb.Organizations
		entity, err := ep.AllTenants().
			Where(
				ep.OrganizationID.Eq(req.TargetOrganizationID),
				ep.BusinessUnitID.Eq(req.BusinessUnitID),
				ep.InternalOrganizationID.Eq(req.SourceOrganizationID),
				ep.Kind.Eq(edi.PartnerKindInternal),
				ep.Status.Eq(domaintypes.StatusActive),
				ep.EnabledForInbound.IsTrue(),
				keel.Exists(org.AllTenants().Where(
					org.ID.EqCol(ep.OrganizationID),
					org.BusinessUnitID.EqCol(ep.BusinessUnitID),
				)),
			).
			Order(ep.CreatedAt.Asc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, reciprocalPartnerErrors.Map(err, req.SourceOrganizationID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListOutboundPartnersByCustomerIDs(
	ctx context.Context,
	req ediports.ListEDIPartnersByCustomerIDsRequest,
) ([]*edi.EDIPartner, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.EDIPartner, error) {
		return r.outboundPartners(ctx, &req, nil)
	})
}

func (r *repository) ListInternalOutboundPartnersByCustomerIDs(
	ctx context.Context,
	req ediports.ListEDIPartnersByCustomerIDsRequest,
) ([]*edi.EDIPartner, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.EDIPartner, error) {
		return r.outboundPartners(
			ctx,
			&req,
			edidb.EDIPartners.Kind.Eq(edi.PartnerKindInternal),
		)
	})
}

func (r *repository) outboundPartners(
	ctx context.Context,
	req *ediports.ListEDIPartnersByCustomerIDsRequest,
	kind keel.Pred,
) ([]*edi.EDIPartner, error) {
	if len(req.CustomerIDs) == 0 {
		return []*edi.EDIPartner{}, nil
	}

	ep := edidb.EDIPartners
	return ep.Where(
		ep.CustomerID.In(req.CustomerIDs...),
		kind,
		ep.Status.Eq(domaintypes.StatusActive),
		ep.EnabledForOutbound.IsTrue(),
	).
		Order(ep.CreatedAt.Asc()).
		All(ctx, r.db.Keel(ctx))
}

func listedPartners(req *ediports.ListEDIPartnersRequest) *keel.Query[edi.EDIPartner] {
	ep := edidb.EDIPartners
	return ep.Where(
		keel.When(req.CustomerID.IsNotNil(), ep.CustomerID.Eq(req.CustomerID)),
		keel.When(req.EnabledForOutbound, ep.EnabledForOutbound.IsTrue()),
		keel.When(req.Status != "", ep.Status.Eq(req.Status)),
	)
}

func withLinks(q *keel.Query[edi.EDIPartner]) *keel.Query[edi.EDIPartner] {
	ep := edidb.EDIPartners
	return q.Load(ep.InternalOrganization(), ep.Connection(), ep.DefaultTransport())
}
