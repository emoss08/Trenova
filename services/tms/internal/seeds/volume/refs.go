package volume

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
	ErrNoOrganization     = errors.New("no organization found; run `task db-reset` first")
	ErrMissingSeedData    = errors.New("development seed data is missing; run `task db-reset` first")
	errNoFreightTemplates = errors.New("no active freight formula templates")
)

type tenantRefs struct {
	orgID         pulid.ID
	buID          pulid.ID
	buCode        string
	orgName       string
	adminID       pulid.ID
	userIDs       []pulid.ID
	stateIDs      map[string]pulid.ID
	stateAbbr     map[pulid.ID]string
	serviceTypes  []pulid.ID
	shipmentTypes []pulid.ID
	accessorials  map[string]*accessorialcharge.AccessorialCharge
	templates     []*formulatemplate.FormulaTemplate
}

func loadTenantRefs(ctx context.Context, db *keel.DB, orgID pulid.ID) (*tenantRefs, error) {
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

	return &tenantRefs{
		orgID:        org.ID,
		buID:         org.BusinessUnitID,
		buCode:       bu.Code,
		orgName:      org.Name,
		stateIDs:     make(map[string]pulid.ID),
		stateAbbr:    make(map[pulid.ID]string),
		accessorials: make(map[string]*accessorialcharge.AccessorialCharge),
	}, nil
}

func loadTenantData(ctx context.Context, db *keel.DB, refs *tenantRefs) error {
	loaders := []func(context.Context, *keel.DB, *tenantRefs) error{
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

func loadUsers(ctx context.Context, db *keel.DB, refs *tenantRefs) error {
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
		refs.userIDs = append(refs.userIDs, user.ID)
		if strings.EqualFold(user.Username, "admin") {
			refs.adminID = user.ID
		}
	}
	if len(refs.userIDs) == 0 {
		return fmt.Errorf("%w: organization %s has no active users", ErrMissingSeedData, refs.orgID)
	}
	if refs.adminID.IsNil() {
		refs.adminID = refs.userIDs[0]
	}

	return nil
}

func loadStates(ctx context.Context, db *keel.DB, refs *tenantRefs) error {
	states, err := usstatedb.UsStates.Query().All(ctx, db)
	if err != nil {
		return fmt.Errorf("load states: %w", err)
	}
	for _, s := range states {
		refs.stateIDs[s.Abbreviation] = s.ID
		refs.stateAbbr[s.ID] = s.Abbreviation
	}
	for i := range cities {
		if _, ok := refs.stateIDs[cities[i].State]; !ok {
			return fmt.Errorf("%w: state %s is not seeded", ErrMissingSeedData, cities[i].State)
		}
	}

	return nil
}

func loadServiceAndShipmentTypes(ctx context.Context, db *keel.DB, refs *tenantRefs) error {
	st := servicetypedb.ServiceTypes
	services, err := st.Pick(st.ID).Where(st.Status.Eq(domaintypes.StatusActive)).All(ctx, db)
	if err != nil {
		return fmt.Errorf("load service types: %w", err)
	}
	for _, s := range services {
		refs.serviceTypes = append(refs.serviceTypes, s.ID)
	}

	sht := shipmenttypedb.ShipmentTypes
	types, err := sht.Pick(sht.ID).Where(sht.Status.Eq(domaintypes.StatusActive)).All(ctx, db)
	if err != nil {
		return fmt.Errorf("load shipment types: %w", err)
	}
	for _, t := range types {
		refs.shipmentTypes = append(refs.shipmentTypes, t.ID)
	}

	if len(refs.serviceTypes) == 0 || len(refs.shipmentTypes) == 0 {
		return fmt.Errorf("%w: no active service or shipment types", ErrMissingSeedData)
	}

	return nil
}

func loadAccessorials(ctx context.Context, db *keel.DB, refs *tenantRefs) error {
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

func loadTemplates(ctx context.Context, db *keel.DB, refs *tenantRefs) error {
	ft := formulatemplatedb.FormulaTemplates
	templates, err := ft.Where(ft.Status.Eq(formulatemplate.StatusActive)).All(ctx, db)
	if err != nil {
		return fmt.Errorf("load formula templates: %w", err)
	}
	if len(templates) == 0 {
		return fmt.Errorf("%w: %w", ErrMissingSeedData, errNoFreightTemplates)
	}
	refs.templates = templates

	return nil
}

func (r *tenantRefs) accessorial(codes ...string) *accessorialcharge.AccessorialCharge {
	for _, code := range codes {
		if c, ok := r.accessorials[strings.ToUpper(code)]; ok {
			return c
		}
	}
	return nil
}
