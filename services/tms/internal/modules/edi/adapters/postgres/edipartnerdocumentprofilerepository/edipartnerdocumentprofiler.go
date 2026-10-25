//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package edipartnerdocumentprofilerepository

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var documentProfileErrors = dberror.Mapping{Entity: "EDIPartnerDocumentProfile"}

var errAmbiguousActiveProfile = errors.New(
	"multiple active EDI document profiles match partner; transaction set and direction are required",
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

func New(p Params) ediports.EDIPartnerDocumentProfileRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-partner-document-profile-repository"),
	}
}

type profileFilter struct {
	transactionSet edi.TransactionSet
	direction      edi.DocumentDirection
	standard       edi.EDIStandard
	status         edi.DocumentStatus
	partnerID      pulid.ID
	search         string
}

func (f *profileFilter) where() []keel.Pred {
	dp := edidb.EDIPartnerDocumentProfiles
	search := strings.TrimSpace(f.search)
	return []keel.Pred{
		keel.When(f.transactionSet != "", dp.TransactionSet.Eq(f.transactionSet)),
		keel.When(f.direction != "", dp.Direction.Eq(f.direction)),
		keel.When(f.standard != "", dp.Standard.Eq(f.standard)),
		keel.When(f.status != "", dp.Status.Eq(f.status)),
		keel.When(f.partnerID.IsNotNil(), dp.EDIPartnerID.Eq(f.partnerID)),
		keel.When(search != "", keel.Or(
			dp.Name.ContainsFold(search),
			dp.Partner().Code.ContainsFold(search),
			dp.Partner().Name.ContainsFold(search),
			dp.DocumentType().Name.ContainsFold(search),
		)),
	}
}

func (r *repository) ListPartnerDocumentProfiles(
	ctx context.Context,
	req *ediports.ListEDIPartnerDocumentProfilesRequest,
) (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error) {
			dp := edidb.EDIPartnerDocumentProfiles
			q := listProfilesQuery(req).Order(dp.CreatedAt.Desc())
			return listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
		},
	)
}

func listProfilesQuery(
	req *ediports.ListEDIPartnerDocumentProfilesRequest,
) *keel.Query[*edi.EDIPartnerDocumentProfile] {
	dp := edidb.EDIPartnerDocumentProfiles
	filter := profileFilter{
		transactionSet: req.TransactionSet,
		direction:      req.Direction,
		standard:       req.Standard,
		status:         req.Status,
		partnerID:      req.PartnerID,
		search:         req.Filter.Query,
	}

	return dp.Where(filter.where()...).Load(
		dp.Partner(),
		dp.DocumentType(),
		dp.Template(),
		dp.PartnerSettingsSchema(),
	)
}

func (r *repository) SelectPartnerDocumentProfileOptions(
	ctx context.Context,
	req *ediports.EDIPartnerDocumentProfileSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error) {
			return listquery.Offset(
				ctx,
				r.db.Keel(ctx),
				profileOptionsQuery(req),
				req.SelectQueryRequest.Pagination,
			)
		},
	)
}

func profileOptionsQuery(
	req *ediports.EDIPartnerDocumentProfileSelectOptionsRequest,
) *keel.Query[*edi.EDIPartnerDocumentProfile] {
	dp := edidb.EDIPartnerDocumentProfiles
	filter := profileFilter{
		transactionSet: req.TransactionSet,
		direction:      req.Direction,
		standard:       req.Standard,
		status:         req.Status,
		partnerID:      req.PartnerID,
		search:         req.SelectQueryRequest.Query,
	}

	return dp.Where(filter.where()...).
		Load(dp.Partner(), dp.Template(), dp.DocumentType()).
		Order(dp.Name.Asc())
}

func (r *repository) GetPartnerDocumentProfileByID(
	ctx context.Context,
	req ediports.GetEDIPartnerDocumentProfileByIDRequest,
) (*edi.EDIPartnerDocumentProfile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDIPartnerDocumentProfile, error) {
		dp := edidb.EDIPartnerDocumentProfiles
		entity, err := dp.Where(dp.ID.Eq(req.ID)).
			Load(
				dp.Partner(),
				dp.DocumentType(),
				dp.Template(),
				dp.TemplateVersion(),
				dp.PartnerSettingsSchema(),
			).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, documentProfileErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetActivePartnerDocumentProfile(
	ctx context.Context,
	req ediports.GetActiveEDIPartnerDocumentProfileRequest,
) (*edi.EDIPartnerDocumentProfile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDIPartnerDocumentProfile, error) {
		dp := edidb.EDIPartnerDocumentProfiles
		entities, err := dp.Where(
			dp.EDIPartnerID.Eq(req.PartnerID),
			dp.Status.Eq(edi.DocumentStatusActive),
			keel.When(req.TransactionSet != "", dp.TransactionSet.Eq(req.TransactionSet)),
			keel.When(req.Direction != "", dp.Direction.Eq(req.Direction)),
		).
			Load(dp.DocumentType(), dp.Template(), dp.PartnerSettingsSchema()).
			Order(dp.CreatedAt.Desc()).
			Limit(2).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, documentProfileErrors.Map(err, req.PartnerID.String())
		}
		if len(entities) == 0 {
			return nil, documentProfileErrors.Map(keel.ErrNotFound, req.PartnerID.String())
		}
		if (req.TransactionSet == "" || req.Direction == "") && len(entities) > 1 {
			return nil, errAmbiguousActiveProfile
		}

		return entities[0], nil
	})
}

func (r *repository) CreatePartnerDocumentProfile(
	ctx context.Context,
	entity *edi.EDIPartnerDocumentProfile,
) (*edi.EDIPartnerDocumentProfile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDIPartnerDocumentProfile, error) {
		if err := edidb.EDIPartnerDocumentProfiles.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, documentProfileErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdatePartnerDocumentProfile(
	ctx context.Context,
	entity *edi.EDIPartnerDocumentProfile,
) (*edi.EDIPartnerDocumentProfile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDIPartnerDocumentProfile, error) {
		dp := edidb.EDIPartnerDocumentProfiles
		err := dp.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			dp.TemplateID,
			dp.TemplateVersionID,
			dp.DocumentTypeID,
			dp.Name,
			dp.Status,
			dp.Direction,
			dp.Standard,
			dp.TransactionSet,
			dp.X12VersionOverride,
			dp.FunctionalGroupID,
			dp.Envelope,
			dp.Acknowledgment,
			dp.ValidationMode,
			dp.PartnerSettings,
			dp.PartnerSettingsSchemaID,
			dp.PartnerSettingsSchemaVersion,
		)
		if err != nil {
			return nil, documentProfileErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}
