package scimrepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/scim"
	"github.com/emoss08/trenova/internal/modules/identity/domain/scim/scimdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func groupPredicates(q *scimports.ListQuery) []keel.Pred {
	sg := scimdb.Groups
	preds := []keel.Pred{sg.DirectoryID.Eq(q.DirectoryID)}
	if q.Filter.Empty() {
		return preds
	}

	for _, condition := range q.Filter.Conditions {
		switch condition.Attribute {
		case scimports.FilterID:
			preds = append(preds, sg.ID.Eq(pulid.ID(condition.Value)))
		case scimports.FilterDisplayName:
			preds = append(preds, sg.DisplayName.Lower().Eq(strings.ToLower(condition.Value)))
		case scimports.FilterExternalID:
			preds = append(preds, sg.ExternalID.Eq(condition.Value))
		case scimports.FilterMember:
			gm := scimdb.GroupMembers
			preds = append(preds, keel.Exists(gm.Where(
				gm.GroupID.EqCol(sg.ID),
				gm.UserID.Eq(pulid.ID(condition.Value)),
			)))
		}
	}

	return preds
}

type listGroupsPage struct {
	groups []*scim.Group
	total  int
}

func (r *repository) ListGroups(
	ctx context.Context,
	q *scimports.ListQuery,
) ([]*scim.Group, int, error) {
	result, err := r.readGroupsPage(ctx, q)
	if err != nil {
		return nil, 0, err
	}

	return result.groups, result.total, nil
}

func (r *repository) readGroupsPage(
	ctx context.Context,
	q *scimports.ListQuery,
) (listGroupsPage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (listGroupsPage, error) {
		sg := scimdb.Groups

		var groups []*scim.Group
		var total keel.RowCount
		if err := keel.Batch(
			ctx,
			r.db.Keel(ctx),
			keel.ListInto(
				sg.Where(groupPredicates(q)...).
					Order(sg.CreatedAt.Asc(), sg.ID.Asc()).
					Offset(q.Offset).
					Limit(q.Limit),
				&groups,
			),
			keel.CountEstimateInto(
				sg.Where(groupPredicates(q)...),
				pagination.ExactTotalUpTo,
				&total,
			),
		); err != nil {
			return listGroupsPage{}, groupErrors.Map(err, q.DirectoryID.String())
		}

		return listGroupsPage{groups: groups, total: int(total.N)}, nil
	})
}

func (r *repository) GetGroup(
	ctx context.Context,
	directoryID, groupID pulid.ID,
) (*scim.Group, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*scim.Group, error) {
		sg := scimdb.Groups
		group, err := sg.Where(sg.DirectoryID.Eq(directoryID), sg.ID.Eq(groupID)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, groupErrors.Map(err, groupID.String())
		}

		return group, nil
	})
}

func (r *repository) CreateGroup(ctx context.Context, group *scim.Group) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := scimdb.Groups.Create(ctx, r.db.Keel(ctx), group); err != nil {
			return groupErrors.Map(err, group.ID.String())
		}

		return nil
	})
}

func (r *repository) SaveGroup(ctx context.Context, group *scim.Group) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		sg := scimdb.Groups
		if err := sg.Save(ctx, r.db.Keel(ctx), group, sg.DisplayName, sg.ExternalID); err != nil {
			return groupErrors.Map(err, group.ID.String())
		}

		return nil
	})
}

func (r *repository) DeleteGroup(ctx context.Context, directoryID, groupID pulid.ID) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		sg := scimdb.Groups
		deleted, err := sg.Where(sg.DirectoryID.Eq(directoryID), sg.ID.Eq(groupID)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return groupErrors.Map(err, groupID.String())
		}

		return dberror.CheckAffected(deleted, groupErrors.Entity, groupID.String())
	})
}

func (r *repository) ListMembers(
	ctx context.Context,
	directoryID, groupID pulid.ID,
	limit int,
) ([]*scimports.MemberRecord, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*scimports.MemberRecord, error) {
		gm := scimdb.GroupMembers
		members, err := gm.Where(gm.DirectoryID.Eq(directoryID), gm.GroupID.Eq(groupID)).
			Load(gm.User()).
			Order(gm.CreatedAt.Asc(), gm.ID.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, memberErrors.Map(err, groupID.String())
		}

		records := make([]*scimports.MemberRecord, 0, len(members))
		for _, member := range members {
			record := &scimports.MemberRecord{UserID: member.UserID}
			if member.User != nil {
				record.Name = member.User.Name
			}
			records = append(records, record)
		}

		return records, nil
	})
}

func (r *repository) AddMembers(ctx context.Context, change *scimports.MemberChange) error {
	if len(change.UserIDs) == 0 {
		return nil
	}

	tenantScope, ok := dbscope.TenantFrom(ctx)
	if !ok {
		return errortypes.NewBusinessError("A SCIM group change needs the directory's tenant")
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		gm := scimdb.GroupMembers
		rows := make([]*scim.GroupMember, 0, len(change.UserIDs))
		for _, userID := range change.UserIDs {
			rows = append(rows, &scim.GroupMember{
				OrganizationID: tenantScope.OrganizationID,
				BusinessUnitID: tenantScope.BusinessUnitID,
				DirectoryID:    change.DirectoryID,
				GroupID:        change.GroupID,
				UserID:         userID,
			})
		}

		if _, err := gm.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			rows,
			gm.OrganizationID,
			gm.BusinessUnitID,
			gm.GroupID,
			gm.UserID,
		); err != nil {
			return memberErrors.Map(err, change.GroupID.String())
		}

		return nil
	})
}

func (r *repository) RemoveMembers(ctx context.Context, change *scimports.MemberChange) error {
	if len(change.UserIDs) == 0 {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		gm := scimdb.GroupMembers
		if _, err := gm.Where(
			gm.DirectoryID.Eq(change.DirectoryID),
			gm.GroupID.Eq(change.GroupID),
			gm.UserID.In(change.UserIDs...),
		).Delete(ctx, r.db.Keel(ctx)); err != nil {
			return memberErrors.Map(err, change.GroupID.String())
		}

		return nil
	})
}

func (r *repository) ClearMembers(ctx context.Context, directoryID, groupID pulid.ID) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		gm := scimdb.GroupMembers
		if _, err := gm.Where(gm.DirectoryID.Eq(directoryID), gm.GroupID.Eq(groupID)).
			Delete(ctx, r.db.Keel(ctx)); err != nil {
			return memberErrors.Map(err, groupID.String())
		}

		return nil
	})
}
