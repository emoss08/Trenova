package tenantpeoplerepository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/passwordreset"
	"github.com/emoss08/trenova/internal/modules/identity/domain/passwordreset/passwordresetdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

const (
	factsReason    = "read whether each shared-server tenant's first administrator is set up and holds a live invite, for the control plane's status report; one statement for the page of tenants"
	emailReason    = "check whether an address is already taken anywhere on a shared server before a tenant is made with it; answers only whether it is"
	reassignReason = "move a removed tenant's member who belongs to another organization onto that organization before the tenant's rows are deleted"
	maxPage        = 500
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

type Result struct {
	fx.Out

	Administrators cloudtenantports.Administrators
	Members        cloudtenantports.Members
}

func New(p Params) Result {
	r := &repository{db: p.DB}
	return Result{Administrators: r, Members: r}
}

func (r *repository) Facts(
	ctx context.Context,
	requests []cloudtenantports.AdministratorRequest,
) (map[pulid.ID]cloudtenantports.AdministratorFacts, error) {
	if len(requests) == 0 {
		return map[pulid.ID]cloudtenantports.AdministratorFacts{}, nil
	}
	userIDs := make([]pulid.ID, 0, len(requests))
	tokens := make([]string, 0, len(requests))
	for _, req := range requests {
		userIDs = append(userIDs, req.UserID)
		if req.InviteToken != "" {
			tokens = append(tokens, req.InviteToken)
		}
	}
	ctx = dbscope.WithSystem(ctx, factsReason)
	return dbtx.Read(ctx, r.db, func(
		ctx context.Context,
	) (map[pulid.ID]cloudtenantports.AdministratorFacts, error) {
		db := r.db.Keel(ctx)
		u := tenantdb.Users
		var users []*tenant.User
		var invites []*passwordreset.PasswordResetToken
		p := passwordresetdb.PasswordResetTokens
		reads := []keel.BatchItem{
			keel.ListInto(u.Where(u.ID.In(userIDs...)).AllTenants().Limit(len(userIDs)), &users),
		}
		if len(tokens) > 0 {
			reads = append(reads, keel.ListInto(
				p.Where(
					p.UserID.In(userIDs...),
					p.TokenHash.In(tokens...),
					p.Purpose.Eq(passwordreset.PasswordResetPurposeInvite),
				).AllTenants().Limit(len(userIDs)*4),
				&invites,
			))
		}
		if err := keel.Batch(ctx, db, reads...); err != nil {
			return nil, fmt.Errorf("read the first administrators: %w", err)
		}
		now := timeutils.NowUnix()
		live := make(map[pulid.ID]string, len(invites))
		for _, inv := range invites {
			if inv.IsRedeemable(now) {
				live[inv.UserID] = inv.TokenHash
			}
		}
		out := make(map[pulid.ID]cloudtenantports.AdministratorFacts, len(requests))
		wanted := make(map[pulid.ID]string, len(requests))
		for _, req := range requests {
			wanted[req.UserID] = req.InviteToken
		}
		for _, usr := range users {
			facts := cloudtenantports.AdministratorFacts{
				UserID: usr.ID,
				SetUp:  !usr.AwaitingAccountSetup(),
			}
			if token, ok := live[usr.ID]; ok && !facts.SetUp && token == wanted[usr.ID] {
				facts.InviteLive = true
				facts.InviteToken = token
			}
			out[usr.ID] = facts
		}
		return out, nil
	})
}

func (r *repository) EmailTaken(ctx context.Context, email string) (bool, error) {
	if tenant.IsReservedEmailAddress(email) {
		return true, nil
	}
	ctx = dbscope.WithSystem(ctx, emailReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		u := tenantdb.Users
		taken, err := u.Where(u.EmailAddress.Lower().Eq(strings.ToLower(strings.TrimSpace(email)))).
			AllTenants().
			Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			return false, fmt.Errorf("check whether an address is taken: %w", err)
		}
		return taken, nil
	})
}

type memberRow struct {
	UserID pulid.ID
}

type otherRow struct {
	UserID pulid.ID
	Total  int64
}

func (r *repository) Page(
	ctx context.Context,
	scope dbscope.Tenant,
	after pulid.ID,
	limit int,
) (*cloudtenantports.MembersPage, error) {
	limit = min(max(limit, 1), maxPage)
	ctx = dbscope.WithSystem(ctx, reassignReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*cloudtenantports.MembersPage, error) {
		db := r.db.Keel(ctx)
		m := tenantdb.OrganizationMemberships
		rows, err := keel.Into[memberRow](
			ctx,
			db,
			m.Where(
				m.OrganizationID.Eq(scope.OrganizationID),
				keel.When(!after.IsNil(), m.UserID.Gt(after)),
			).
				AllTenants().
				Order(m.UserID.Asc()).
				Limit(limit+1).
				Pick(m.UserID),
		)
		if err != nil {
			return nil, fmt.Errorf("list the tenant's members: %w", err)
		}
		page := &cloudtenantports.MembersPage{}
		if len(rows) > limit {
			rows = rows[:limit]
			page.Next = rows[limit-1].UserID
		}
		if len(rows) == 0 {
			return page, nil
		}
		ids := make([]pulid.ID, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.UserID)
		}
		others, err := keel.Into[otherRow](
			ctx,
			db,
			m.Where(m.UserID.In(ids...), m.OrganizationID.Ne(scope.OrganizationID)).
				AllTenants().
				Group(m.UserID).
				Pick(m.UserID, keel.CastInt64(keel.Count())),
		)
		if err != nil {
			return nil, fmt.Errorf("count the members' other organizations: %w", err)
		}
		byUser := make(map[pulid.ID]int64, len(others))
		for _, o := range others {
			byUser[o.UserID] = o.Total
		}
		page.Members = make([]cloudtenantports.Member, 0, len(ids))
		for _, id := range ids {
			page.Members = append(page.Members, cloudtenantports.Member{
				UserID:           id,
				OtherMemberships: int(byUser[id]),
			})
		}
		return page, nil
	})
}

func (r *repository) Reassign(
	ctx context.Context,
	scope dbscope.Tenant,
	userID pulid.ID,
) (bool, error) {
	ctx = dbscope.WithSystem(ctx, reassignReason)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		db := r.db.Keel(ctx)
		m := tenantdb.OrganizationMemberships
		other, err := m.Where(m.UserID.Eq(userID), m.OrganizationID.Ne(scope.OrganizationID)).
			AllTenants().
			Order(m.IsDefault.Desc(), m.JoinedAt.Asc()).
			First(ctx, db)
		if errors.Is(err, keel.ErrNotFound) {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("read the member's other organization: %w", err)
		}
		u := tenantdb.Users
		if _, err = u.Where(
			u.ID.Eq(userID),
			keel.Or(
				u.CurrentOrganizationID.Eq(scope.OrganizationID),
				u.BusinessUnitID.Eq(scope.BusinessUnitID),
			),
		).AllTenants().Update(
			ctx,
			db,
			u.CurrentOrganizationID.To(other.OrganizationID),
			u.BusinessUnitID.To(other.BusinessUnitID),
			u.UpdatedAt.To(timeutils.NowUnix()),
		); err != nil {
			return false, fmt.Errorf("move the member onto their other organization: %w", err)
		}
		return true, nil
	})
}
