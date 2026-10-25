package accountingsyncrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync/accountingsyncdb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultReferenceSearch = 25
	maxReferenceSearch     = 100
)

type ReferenceParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type referenceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewReferenceRepository(
	p ReferenceParams,
) accountingsyncports.AccountingReferenceObjectRepository {
	return &referenceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-reference-repository"),
	}
}

func (r *referenceRepository) Upsert(
	ctx context.Context,
	req *accountingsyncports.UpsertAccountingReferenceObjectsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(req.Objects) == 0 {
			return nil
		}

		for _, obj := range req.Objects {
			obj.OrganizationID = req.TenantInfo.OrgID
			obj.BusinessUnitID = req.TenantInfo.BuID
			obj.ConnectionID = req.ConnectionID
			obj.LastSeenAt = req.SeenAt
			obj.RemovedAt = nil
		}

		ro := accountingsyncdb.AccountingReferenceObjects
		_, err := ro.Upsert(
			ctx,
			r.db.Keel(ctx),
			req.Objects,
			keel.OnConflict(
				ro.OrganizationID,
				ro.BusinessUnitID,
				ro.ConnectionID,
				ro.Kind,
				ro.ExternalID,
			).
				Update(
					ro.Name,
					ro.SearchName,
					ro.FullyQualifiedName,
					ro.Number,
					ro.Description,
					ro.Classification,
					ro.AccountType,
					ro.AccountSubType,
					ro.ItemType,
					ro.SubType,
					ro.ParentExternalID,
					ro.Active,
					ro.CurrencyCode,
					ro.SyncToken,
					ro.CompanyName,
					ro.Email,
					ro.AddressLine1,
					ro.City,
					ro.State,
					ro.PostalCode,
					ro.IncomeAccountExternalID,
					ro.DueDays,
					ro.Is1099,
					ro.ProviderUpdatedAt,
					ro.LastSeenAt,
					ro.UpdatedAt,
				).
				Set(ro.RemovedAt.ToNull()),
		)
		if err != nil {
			return fmt.Errorf("upsert accounting reference objects: %w", err)
		}

		return nil
	})
}

func (r *referenceRepository) MarkRemoved(
	ctx context.Context,
	req *accountingsyncports.MarkAccountingReferencesRemovedRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		if len(req.ExternalIDs) == 0 {
			return 0, nil
		}

		ro := accountingsyncdb.AccountingReferenceObjects
		return ro.Where(
			ro.ConnectionID.Eq(req.ConnectionID),
			ro.Kind.Eq(req.Kind),
			ro.ExternalID.In(req.ExternalIDs...),
			ro.RemovedAt.IsNull(),
		).Update(ctx, r.db.Keel(ctx), ro.RemovedAt.To(req.At), ro.UpdatedAt.To(req.At))
	})
}

func (r *referenceRepository) MarkRemovedUnseen(
	ctx context.Context,
	req *accountingsyncports.MarkAccountingReferenceRemovedRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		ro := accountingsyncdb.AccountingReferenceObjects
		return ro.Where(
			ro.ConnectionID.Eq(req.ConnectionID),
			ro.Kind.Eq(req.Kind),
			ro.LastSeenAt.Lt(req.SeenBefore),
			ro.RemovedAt.IsNull(),
		).Update(ctx, r.db.Keel(ctx), ro.RemovedAt.To(req.At), ro.UpdatedAt.To(req.At))
	})
}

func (r *referenceRepository) ListByKind(
	ctx context.Context,
	req *accountingsyncports.ListAccountingReferenceObjectsRequest,
) ([]*accountingsync.AccountingReferenceObject, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingReferenceObject, error) {
			ro := accountingsyncdb.AccountingReferenceObjects
			return ro.Where(
				ro.ConnectionID.Eq(req.ConnectionID),
				ro.Kind.Eq(req.Kind),
				keel.When(req.UsableOnly, usableReference()),
			).
				Order(ro.SearchName.Asc(), ro.ExternalID.Asc()).
				All(ctx, r.db.Keel(ctx))
		},
	)
}

func (r *referenceRepository) Search(
	ctx context.Context,
	req *accountingsyncports.SearchAccountingReferenceObjectsRequest,
) ([]*accountingsync.AccountingReferenceObject, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingReferenceObject, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultReferenceSearch
			}
			limit = min(limit, maxReferenceSearch)

			ro := accountingsyncdb.AccountingReferenceObjects
			q := ro.Where(
				ro.ConnectionID.Eq(req.ConnectionID),
				ro.Kind.Eq(req.Kind),
				keel.When(req.UsableOnly, usableReference()),
			).Limit(limit)

			if term := stringutils.NormalizeName(req.Query); term != "" {
				prefix := stringutils.EscapeLikePattern(term) + "%"
				q = q.Where(keel.Or(
					ro.SearchName.Contains(term),
					ro.Number.ILike(prefix),
					ro.ExternalID.Eq(strings.TrimSpace(req.Query)),
				)).Order(keel.Asc(keel.Raw("CASE WHEN ? LIKE ? THEN 0 ELSE 1 END", ro.SearchName, prefix)))
			}

			return q.Order(ro.SearchName.Asc(), ro.ExternalID.Asc()).List(ctx, r.db.Keel(ctx))
		},
	)
}

func (r *referenceRepository) GetByExternalIDs(
	ctx context.Context,
	req *accountingsyncports.GetAccountingReferenceObjectsRequest,
) ([]*accountingsync.AccountingReferenceObject, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingReferenceObject, error) {
			if len(req.ExternalIDs) == 0 {
				return []*accountingsync.AccountingReferenceObject{}, nil
			}

			ro := accountingsyncdb.AccountingReferenceObjects
			return ro.Where(
				ro.ConnectionID.Eq(req.ConnectionID),
				ro.ExternalID.In(req.ExternalIDs...),
				keel.When(req.Kind != "", ro.Kind.Eq(req.Kind)),
			).All(ctx, r.db.Keel(ctx))
		},
	)
}

func usableReference() keel.Pred {
	ro := accountingsyncdb.AccountingReferenceObjects
	return keel.And(
		ro.Active.IsTrue(),
		ro.RemovedAt.IsNull(),
		keel.Or(
			ro.ItemType.IsNull(),
			ro.ItemType.NotIn(accountingsync.ItemTypeCategory, accountingsync.ItemTypeGroup),
		),
	)
}
