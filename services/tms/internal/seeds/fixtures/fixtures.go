package fixtures

import (
	"context"
	"crypto/md5" //nolint:gosec // identifier derivation, not security
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/shared/pulid"
)

var Seeds = seed.NewRegistry()

var errFixtureMissing = errors.New("a fixture this seed builds on is missing")

func backdate[T any](
	ctx context.Context,
	db *keel.DB,
	h keel.Handle[*T],
	ids []string,
	ats []int64,
) error {
	if len(ids) == 0 {
		return nil
	}

	t := h.Ref.Table
	id := keel.Col{Ref: h.Ref, Index: t.PK[0]}
	created := keel.Col{Ref: h.Ref, Index: t.AutoCreate}
	values := keel.Unnest("backdated", []string{"id", "at"}, ids, ats)
	sets := []keel.Assignment{created.ToExpr(values.Col("at"))}
	if t.AutoUpdate >= 0 {
		sets = append(sets, keel.Col{Ref: h.Ref, Index: t.AutoUpdate}.ToExpr(values.Col("at")))
	}
	if t.Version >= 0 {
		version := keel.Col{Ref: h.Ref, Index: t.Version}
		sets = append(sets, version.ToExpr(version))
	}

	q := h.Query()
	if len(t.Tenant) > 0 {
		q = h.AllTenants()
	}
	if _, err := q.With(values).
		Join(values, values.Col("id").EqCol(id)).
		Where(keel.True()).
		Update(ctx, db, sets...); err != nil {
		return fmt.Errorf("backdate %s: %w", t.Name, err)
	}

	return nil
}

func ptrs[T any](rows []T) []*T {
	out := make([]*T, len(rows))
	for i := range rows {
		out[i] = &rows[i]
	}

	return out
}

func deterministicID(prefix, key string) pulid.ID {
	sum := md5.Sum([]byte(key)) //nolint:gosec // identifier derivation, not security
	return pulid.ID(prefix + strings.ToUpper(hex.EncodeToString(sum[:]))[:26])
}

var ErrKnownCredentialsOutsideDevelopment = errors.New(
	"refusing to create accounts with publicly known passwords outside development and test",
)

var (
	defaultOrgKey  = seed.Key[*tenant.Organization]("default-organization")
	defaultScacs   = []string{"DFLT", "TRNV"}
	adminUsername  = "admin"
	secondOrgScac  = "TTNV"
	defaultBUCode  = "DEFAULT"
	defaultOrgName = "Trenova Logistics"
)

func requireDevelopment(r *seed.Run) error {
	switch r.Environment {
	case seed.Production, seed.Staging:
		return ErrKnownCredentialsOutsideDevelopment
	default:
		return nil
	}
}

func defaultOrganization(ctx context.Context, r *seed.Run) (*tenant.Organization, error) {
	if org, ok := seed.Get(r, defaultOrgKey); ok {
		return org, nil
	}

	o := tenantdb.Organizations
	org, err := o.AllTenants().
		Where(o.ScacCode.In(defaultScacs...)).
		Order(o.CreatedAt.Asc(), o.ID.Asc()).
		First(ctx, r.DB)
	if err != nil {
		return nil, fmt.Errorf("get default organization: %w", err)
	}
	seed.Put(r, defaultOrgKey, org)

	return org, nil
}

func organizationByScac(
	ctx context.Context,
	db *keel.DB,
	scac string,
) (*tenant.Organization, error) {
	o := tenantdb.Organizations
	org, err := o.AllTenants().Where(o.ScacCode.Eq(scac)).First(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get organization %s: %w", scac, err)
	}

	return org, nil
}

func userByUsername(ctx context.Context, db *keel.DB, username string) (*tenant.User, error) {
	u := tenantdb.Users
	user, err := u.AllTenants().Where(u.Username.Eq(username)).First(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get user %s: %w", username, err)
	}

	return user, nil
}

func adminUser(ctx context.Context, db *keel.DB) (*tenant.User, error) {
	return userByUsername(ctx, db, adminUsername)
}

func stateByAbbreviation(
	ctx context.Context,
	db *keel.DB,
	abbreviation string,
) (*usstate.UsState, error) {
	s := usstatedb.UsStates
	state, err := s.Where(s.Abbreviation.Eq(abbreviation)).First(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get state %s: %w", abbreviation, err)
	}

	return state, nil
}
