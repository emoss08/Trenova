package scimrepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/scim"
	"github.com/emoss08/trenova/internal/modules/identity/domain/scim/scimdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func userPredicates(q *scimports.ListQuery) []keel.Pred {
	su := scimdb.Users
	preds := []keel.Pred{su.DirectoryID.Eq(q.DirectoryID)}
	if q.Filter.Empty() {
		return preds
	}

	for _, condition := range q.Filter.Conditions {
		switch condition.Attribute {
		case scimports.FilterID:
			preds = append(preds, su.UserID.Eq(pulid.ID(condition.Value)))
		case scimports.FilterUserName:
			preds = append(preds, su.UserName.Lower().Eq(strings.ToLower(condition.Value)))
		case scimports.FilterExternalID:
			preds = append(preds, su.ExternalID.Eq(condition.Value))
		case scimports.FilterActive:
			preds = append(preds, su.Active.Eq(condition.Value == "true"))
		case scimports.FilterEmail:
			u := tenantdb.Users
			preds = append(preds, keel.Exists(u.Where(
				u.ID.EqCol(su.UserID),
				u.EmailAddress.Lower().Eq(strings.ToLower(condition.Value)),
			)))
		}
	}

	return preds
}

func (r *repository) ListUsers(
	ctx context.Context,
	q *scimports.ListQuery,
) ([]*scimports.UserRecord, int, error) {
	type page struct {
		records []*scimports.UserRecord
		total   int
	}

	result, err := dbtx.Read(ctx, r.db, func(ctx context.Context) (page, error) {
		su := scimdb.Users
		base := su.Where(userPredicates(q)...)

		var links []*scim.User
		var total keel.RowCount
		if err := keel.Batch(
			ctx,
			r.db.Keel(ctx),
			keel.ListInto(
				su.Where(userPredicates(q)...).
					Load(su.User()).
					Order(su.CreatedAt.Asc(), su.ID.Asc()).
					Offset(q.Offset).
					Limit(q.Limit),
				&links,
			),
			keel.CountEstimateInto(base, pagination.ExactTotalUpTo, &total),
		); err != nil {
			return page{}, userErrors.Map(err, q.DirectoryID.String())
		}

		return page{records: userRecords(links), total: int(total.N)}, nil
	})
	if err != nil {
		return nil, 0, err
	}

	return result.records, result.total, nil
}

func userRecords(links []*scim.User) []*scimports.UserRecord {
	records := make([]*scimports.UserRecord, 0, len(links))
	for _, link := range links {
		records = append(records, &scimports.UserRecord{Link: link, User: link.User})
	}

	return records
}

func (r *repository) GetUser(
	ctx context.Context,
	directoryID, userID pulid.ID,
) (*scimports.UserRecord, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*scimports.UserRecord, error) {
		su := scimdb.Users
		link, err := su.Where(su.DirectoryID.Eq(directoryID), su.UserID.Eq(userID)).
			Load(su.User()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, userErrors.Map(err, userID.String())
		}

		return &scimports.UserRecord{Link: link, User: link.User}, nil
	})
}

func (r *repository) FindMember(ctx context.Context, emailAddress string) (*tenant.User, error) {
	tenantScope, ok := dbscope.TenantFrom(ctx)
	if !ok {
		return nil, errortypes.NewBusinessError("A SCIM lookup needs the directory's tenant")
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenant.User, error) {
		u := tenantdb.Users
		m := tenantdb.OrganizationMemberships

		user, err := u.Where(
			u.EmailAddress.Lower().Eq(strings.ToLower(emailAddress)),
			keel.Or(
				keel.Exists(m.Where(m.UserID.EqCol(u.ID))),
				u.CurrentOrganizationID.Eq(tenantScope.OrganizationID),
			),
		).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, userErrors.Map(err, emailAddress)
		}

		return user, nil
	})
}

func (r *repository) CreateUser(ctx context.Context, link *scim.User) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := scimdb.Users.Create(ctx, r.db.Keel(ctx), link); err != nil {
			return userErrors.Map(err, link.UserID.String())
		}

		return nil
	})
}

func (r *repository) SaveUser(ctx context.Context, link *scim.User) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		su := scimdb.Users
		if err := su.Save(
			ctx,
			r.db.Keel(ctx),
			link,
			su.UserName,
			su.ExternalID,
			su.Active,
		); err != nil {
			return userErrors.Map(err, link.UserID.String())
		}

		return nil
	})
}

func (r *repository) DeleteUser(ctx context.Context, directoryID, userID pulid.ID) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		su := scimdb.Users
		deleted, err := su.Where(su.DirectoryID.Eq(directoryID), su.UserID.Eq(userID)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return userErrors.Map(err, userID.String())
		}

		return dberror.CheckAffected(deleted, userErrors.Entity, userID.String())
	})
}

func (r *repository) CountUsers(
	ctx context.Context,
	directoryID pulid.ID,
	userIDs []pulid.ID,
) (int, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		su := scimdb.Users
		count, err := su.Where(su.DirectoryID.Eq(directoryID), su.UserID.In(userIDs...)).
			Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, userErrors.Map(err, directoryID.String())
		}

		return int(count), nil
	})
}
