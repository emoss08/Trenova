package volumegen

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype/servicetypedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype/shipmenttypedb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	ErrNoOrganization  = errors.New("no organization found; run `task db-reset` first")
	ErrMissingSeedData = errors.New(
		"development seed data is missing; run `task db-reset` first",
	)
	errNoFreightTemplates = errors.New("no active freight formula templates")
)

type TenantRefs struct {
	OrgID         pulid.ID
	BuID          pulid.ID
	BuCode        string
	OrgName       string
	AdminID       pulid.ID
	UserIDs       []pulid.ID
	StateIDs      map[string]pulid.ID
	StateAbbr     map[pulid.ID]string
	ServiceTypes  []pulid.ID
	ShipmentTypes []pulid.ID
	accessorials  map[string]*accessorialcharge.AccessorialCharge
	Templates     []*formulatemplate.FormulaTemplate
}

func LoadTenantRefs(ctx context.Context, db *keel.DB, orgID pulid.ID) (*TenantRefs, error) {
	o := tenantdb.Organizations
	q := o.AllTenants()
	if orgID.IsNotNil() {
		q = q.Where(o.ID.Eq(orgID))
	} else {
		q = q.Where(o.ScacCode.In("DFLT", "TRNV")).Order(o.CreatedAt.Asc(), o.ID.Asc())
	}
	org, err := q.First(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoOrganization, err)
	}

	b := tenantdb.BusinessUnits
	bu, err := b.Where(b.ID.Eq(org.BusinessUnitID)).One(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("load business unit: %w", err)
	}

	return &TenantRefs{
		OrgID:        org.ID,
		BuID:         org.BusinessUnitID,
		BuCode:       bu.Code,
		OrgName:      org.Name,
		StateIDs:     make(map[string]pulid.ID),
		StateAbbr:    make(map[pulid.ID]string),
		accessorials: make(map[string]*accessorialcharge.AccessorialCharge),
	}, nil
}

func LoadTenantData(ctx context.Context, db *keel.DB, refs *TenantRefs) error {
	loaders := []func(context.Context, *keel.DB, *TenantRefs) error{
		loadUsers,
		loadStates,
		loadServiceAndShipmentTypes,
		loadAccessorials,
		loadTemplates,
	}
	for _, load := range loaders {
		if err := load(ctx, db, refs); err != nil {
			return err
		}
	}

	return nil
}

func loadUsers(ctx context.Context, db *keel.DB, refs *TenantRefs) error {
	m := tenantdb.OrganizationMemberships
	memberships, err := m.Pick(m.UserID).All(ctx, db)
	if err != nil {
		return fmt.Errorf("load memberships: %w", err)
	}
	memberIDs := make([]pulid.ID, 0, len(memberships))
	for _, membership := range memberships {
		memberIDs = append(memberIDs, membership.UserID)
	}

	u := tenantdb.Users
	users, err := u.Where(
		u.ID.In(memberIDs...),
		u.Status.Eq(domaintypes.StatusActive),
	).Order(u.CreatedAt.Asc(), u.ID.Asc()).All(ctx, db)
	if err != nil {
		return fmt.Errorf("load users: %w", err)
	}
	for _, user := range users {
		refs.UserIDs = append(refs.UserIDs, user.ID)
		if strings.EqualFold(user.Username, "admin") {
			refs.AdminID = user.ID
		}
	}
	if len(refs.UserIDs) == 0 {
		return fmt.Errorf("%w: organization %s has no active users", ErrMissingSeedData, refs.OrgID)
	}
	if refs.AdminID.IsNil() {
		refs.AdminID = refs.UserIDs[0]
	}

	return nil
}

func loadStates(ctx context.Context, db *keel.DB, refs *TenantRefs) error {
	states, err := usstatedb.UsStates.Query().All(ctx, db)
	if err != nil {
		return fmt.Errorf("load states: %w", err)
	}
	for _, s := range states {
		refs.StateIDs[s.Abbreviation] = s.ID
		refs.StateAbbr[s.ID] = s.Abbreviation
	}
	for i := range Cities {
		if _, ok := refs.StateIDs[Cities[i].State]; !ok {
			return fmt.Errorf("%w: state %s is not seeded", ErrMissingSeedData, Cities[i].State)
		}
	}

	return nil
}

func loadServiceAndShipmentTypes(ctx context.Context, db *keel.DB, refs *TenantRefs) error {
	st := servicetypedb.ServiceTypes
	services, err := st.Pick(st.ID).Where(st.Status.Eq(domaintypes.StatusActive)).All(ctx, db)
	if err != nil {
		return fmt.Errorf("load service types: %w", err)
	}
	for _, s := range services {
		refs.ServiceTypes = append(refs.ServiceTypes, s.ID)
	}

	sht := shipmenttypedb.ShipmentTypes
	types, err := sht.Pick(sht.ID).Where(sht.Status.Eq(domaintypes.StatusActive)).All(ctx, db)
	if err != nil {
		return fmt.Errorf("load shipment types: %w", err)
	}
	for _, t := range types {
		refs.ShipmentTypes = append(refs.ShipmentTypes, t.ID)
	}

	if len(refs.ServiceTypes) == 0 || len(refs.ShipmentTypes) == 0 {
		return fmt.Errorf("%w: no active service or shipment types", ErrMissingSeedData)
	}

	return nil
}

func loadAccessorials(ctx context.Context, db *keel.DB, refs *TenantRefs) error {
	ac := accessorialchargedb.AccessorialCharges
	charges, err := ac.Where(ac.Status.Eq(domaintypes.StatusActive)).All(ctx, db)
	if err != nil {
		return fmt.Errorf("load accessorial charges: %w", err)
	}
	for _, c := range charges {
		refs.accessorials[strings.ToUpper(c.Code)] = c
	}

	return nil
}

func loadTemplates(ctx context.Context, db *keel.DB, refs *TenantRefs) error {
	ft := formulatemplatedb.FormulaTemplates
	templates, err := ft.Where(ft.Status.Eq(formulatemplate.StatusActive)).All(ctx, db)
	if err != nil {
		return fmt.Errorf("load formula templates: %w", err)
	}
	if len(templates) == 0 {
		return fmt.Errorf("%w: %w", ErrMissingSeedData, errNoFreightTemplates)
	}
	refs.Templates = templates

	return nil
}

func (r *TenantRefs) Accessorial(codes ...string) *accessorialcharge.AccessorialCharge {
	for _, code := range codes {
		if c, ok := r.accessorials[strings.ToUpper(code)]; ok {
			return c
		}
	}
	return nil
}
