//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package edicommunicationprofilerepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var profileErrors = dberror.Mapping{Entity: "EDICommunicationProfile"}

var pollingMethods = []edi.ConnectionMethod{edi.ConnectionMethodSFTP, edi.ConnectionMethodVAN}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) ediports.EDICommunicationProfileRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-communication-profile-repository"),
	}
}

func (r *repository) ListProfiles(
	ctx context.Context,
	req *ediports.ListEDICommunicationProfilesRequest,
) (*pagination.ListResult[*edi.EDICommunicationProfile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDICommunicationProfile], error) {
			cp := edidb.EDICommunicationProfiles
			return listquery.List(
				ctx,
				r.db.Keel(ctx),
				cp.Query(),
				&listquery.Request[edi.EDICommunicationProfile]{
					Filter: req.Filter,
					Shape:  withPartner,
					Then:   []keel.Order{cp.CreatedAt.Desc()},
				},
			)
		},
	)
}

func (r *repository) SelectProfileOptions(
	ctx context.Context,
	req *ediports.EDICommunicationProfileSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDICommunicationProfile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDICommunicationProfile], error) {
			search := *req.SelectQueryRequest
			search.Query = strings.TrimSpace(search.Query)

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				profileOptionsQuery(req),
				&search,
				profileOptionsSearch,
			)
		},
	)
}

func profileOptionsQuery(
	req *ediports.EDICommunicationProfileSelectOptionsRequest,
) *keel.Query[edi.EDICommunicationProfile] {
	cp := edidb.EDICommunicationProfiles
	return cp.Pick(
		cp.ID,
		cp.BusinessUnitID,
		cp.OrganizationID,
		cp.EDIConnectionID,
		cp.EDIPartnerID,
		cp.Method,
		cp.Status,
		cp.Name,
		cp.Description,
	).Where(
		keel.When(req.Status != "", cp.Status.Eq(req.Status)),
		keel.When(req.Method != "", cp.Method.Eq(req.Method)),
		keel.When(req.PartnerID.IsNotNil(), cp.EDIPartnerID.Eq(req.PartnerID)),
	).Order(cp.Name.Asc())
}

func profileOptionsSearch(s string) keel.Pred {
	cp := edidb.EDICommunicationProfiles
	return keel.Or(
		cp.Name.ContainsFold(s),
		cp.Description.ContainsFold(s),
		listquery.TextContainsFold(cp.Method, s),
	)
}

func (r *repository) GetProfileByID(
	ctx context.Context,
	req ediports.GetEDICommunicationProfileByIDRequest,
) (*edi.EDICommunicationProfile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDICommunicationProfile, error) {
		cp := edidb.EDICommunicationProfiles
		entity, err := cp.Where(cp.ID.Eq(req.ID)).Load(cp.Partner()).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, profileErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListProfilesByPartner(
	ctx context.Context,
	req ediports.ListEDICommunicationProfilesByPartnerRequest,
) ([]*edi.EDICommunicationProfile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.EDICommunicationProfile, error) {
		cp := edidb.EDICommunicationProfiles
		return cp.Where(cp.EDIPartnerID.Eq(req.PartnerID)).
			Order(cp.Status.Asc(), cp.CreatedAt.Asc(), cp.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) GetActiveProfileByPartner(
	ctx context.Context,
	req ediports.GetActiveEDICommunicationProfileByPartnerRequest,
) (*edi.EDICommunicationProfile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDICommunicationProfile, error) {
		cp := edidb.EDICommunicationProfiles
		method := cp.Method.Eq(req.Method)
		if len(req.Methods) > 0 {
			method = cp.Method.In(req.Methods...)
		}

		entity, err := cp.Where(
			cp.EDIPartnerID.Eq(req.PartnerID),
			cp.Status.Eq(domaintypes.StatusActive),
			method,
		).First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, profileErrors.Map(err, req.PartnerID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListInboundPollingProfiles(
	ctx context.Context,
) ([]*edi.EDICommunicationProfile, error) {
	ctx = dbscope.WithSystem(ctx, "list inbound EDI polling profiles across every organization")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.EDICommunicationProfile, error) {
		cp := edidb.EDICommunicationProfiles
		return cp.AllTenants().
			Where(pollingProfile()...).
			Order(cp.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) RecordInboundPollOutcome(
	ctx context.Context,
	req ediports.RecordEDIProfilePollOutcomeRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cp := edidb.EDICommunicationProfiles
		sets := []keel.Assignment{
			cp.LastPollAttemptAt.To(req.PolledAt),
			cp.Version.ToExpr(cp.Version),
			cp.UpdatedAt.ToExpr(cp.UpdatedAt),
		}
		if req.Success {
			sets = append(sets, cp.LastPollSuccessAt.To(req.PolledAt), cp.LastPollError.ToNull())
		} else {
			sets = append(sets, cp.LastPollError.To(req.Error))
		}

		_, err := cp.Where(cp.ID.Eq(req.ProfileID)).Update(ctx, r.db.Keel(ctx), sets...)
		return err
	})
}

func (r *repository) CountStaleInboundPollingProfiles(
	ctx context.Context,
	staleBefore int64,
) (int64, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"count stale inbound polling profiles instance-wide for the health check",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		cp := edidb.EDICommunicationProfiles
		return cp.AllTenants().
			Where(pollingProfile()...).
			Where(keel.Raw("COALESCE(?, ?) < ?", cp.LastPollSuccessAt, cp.CreatedAt, staleBefore)).
			Count(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) GetActiveAS2ProfileByIdentifiers(
	ctx context.Context,
	req ediports.GetActiveAS2ProfileByIdentifiersRequest,
) (*edi.EDICommunicationProfile, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"resolve an inbound AS2 message's identifier pair before its tenant is known",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDICommunicationProfile, error) {
		cp := edidb.EDICommunicationProfiles
		entity, err := cp.AllTenants().
			Where(
				cp.Status.Eq(domaintypes.StatusActive),
				cp.Method.Eq(edi.ConnectionMethodAS2),
				cp.EDIPartnerID.IsNotNull(),
				keel.Raw("TRIM(?) = ?", cp.Config.Text("localAS2Id"), req.LocalAS2ID),
				keel.Raw("TRIM(?) = ?", cp.Config.Text("partnerAS2Id"), req.PartnerAS2ID),
			).
			Order(cp.UpdatedAt.Desc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, profileErrors.Map(err, req.LocalAS2ID)
		}

		return entity, nil
	})
}

func (r *repository) CreateProfile(
	ctx context.Context,
	entity *edi.EDICommunicationProfile,
) (*edi.EDICommunicationProfile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDICommunicationProfile, error) {
		if err := edidb.EDICommunicationProfiles.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, profileErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateProfile(
	ctx context.Context,
	entity *edi.EDICommunicationProfile,
) (*edi.EDICommunicationProfile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDICommunicationProfile, error) {
		cp := edidb.EDICommunicationProfiles
		err := cp.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			cp.EDIConnectionID,
			cp.EDIPartnerID,
			cp.Method,
			cp.Status,
			cp.Name,
			cp.Description,
			cp.Config,
			cp.EncryptedSecrets,
		)
		if err != nil {
			return nil, profileErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListProfilesCursor(
	ctx context.Context,
	req *ediports.ListEDICommunicationProfilesRequest,
) (*pagination.CursorListResult[*edi.EDICommunicationProfile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*edi.EDICommunicationProfile], error) {
			return listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				edidb.EDICommunicationProfiles.Query(),
				&listquery.Request[edi.EDICommunicationProfile]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape:  withPartner,
				},
			)
		},
	)
}

func withPartner(
	q *keel.Query[edi.EDICommunicationProfile],
) *keel.Query[edi.EDICommunicationProfile] {
	return q.Load(edidb.EDICommunicationProfiles.Partner())
}

func pollingProfile() []keel.Pred {
	cp := edidb.EDICommunicationProfiles
	return []keel.Pred{
		cp.Status.Eq(domaintypes.StatusActive),
		cp.Method.In(pollingMethods...),
		cp.EDIPartnerID.IsNotNull(),
		keel.Raw("COALESCE(TRIM(?), '') <> ''", cp.Config.Text("inboundDirectory")),
	}
}
