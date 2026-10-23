package volumeseed

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

var (
	ErrNoOrganization     = errors.New("no organization found; run `task db-seed` first")
	ErrMissingSeedData    = errors.New("development seed data is missing; run `task db-seed` first")
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

func loadTenantRefs(ctx context.Context, db bun.IDB, orgID pulid.ID) (*tenantRefs, error) {
	org := new(tenant.Organization)
	q := db.NewSelect().Model(org).Limit(1)
	if orgID.IsNotNil() {
		q = q.Where("org.id = ?", orgID)
	} else {
		q = q.Where("org.scac_code IN (?)", bun.List([]string{"DFLT", "TRNV"})).
			OrderExpr("org.created_at ASC")
	}
	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoOrganization, err)
	}

	bu := new(tenant.BusinessUnit)
	if err := db.NewSelect().Model(bu).Where("bu.id = ?", org.BusinessUnitID).Scan(ctx); err != nil {
		return nil, fmt.Errorf("load business unit: %w", err)
	}

	refs := &tenantRefs{
		orgID:        org.ID,
		buID:         org.BusinessUnitID,
		buCode:       bu.Code,
		orgName:      org.Name,
		stateIDs:     make(map[string]pulid.ID),
		stateAbbr:    make(map[pulid.ID]string),
		accessorials: make(map[string]*accessorialcharge.AccessorialCharge),
	}

	loaders := []func(context.Context, bun.IDB, *tenantRefs) error{
		loadUsers,
		loadStates,
		loadServiceAndShipmentTypes,
		loadAccessorials,
		loadTemplates,
	}
	for _, load := range loaders {
		if err := load(ctx, db, refs); err != nil {
			return nil, err
		}
	}

	return refs, nil
}

func loadUsers(ctx context.Context, db bun.IDB, refs *tenantRefs) error {
	var users []*tenant.User
	if err := db.NewSelect().Model(&users).
		Join("JOIN user_organization_memberships AS uom ON uom.user_id = usr.id").
		Where("uom.organization_id = ?", refs.orgID).
		Where("usr.status = ?", domaintypes.StatusActive).
		Order("usr.created_at").
		Scan(ctx); err != nil {
		return fmt.Errorf("load users: %w", err)
	}
	for _, u := range users {
		refs.userIDs = append(refs.userIDs, u.ID)
		if strings.EqualFold(u.Username, "admin") {
			refs.adminID = u.ID
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

func loadStates(ctx context.Context, db bun.IDB, refs *tenantRefs) error {
	var states []*usstate.UsState
	if err := db.NewSelect().Model(&states).Scan(ctx); err != nil {
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

func loadServiceAndShipmentTypes(ctx context.Context, db bun.IDB, refs *tenantRefs) error {
	var services []*servicetype.ServiceType
	if err := db.NewSelect().Model(&services).
		Where("organization_id = ?", refs.orgID).
		Where("status = ?", domaintypes.StatusActive).
		Scan(ctx); err != nil {
		return fmt.Errorf("load service types: %w", err)
	}
	for _, s := range services {
		refs.serviceTypes = append(refs.serviceTypes, s.ID)
	}

	var types []*shipmenttype.ShipmentType
	if err := db.NewSelect().Model(&types).
		Where("organization_id = ?", refs.orgID).
		Where("status = ?", domaintypes.StatusActive).
		Scan(ctx); err != nil {
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

func loadAccessorials(ctx context.Context, db bun.IDB, refs *tenantRefs) error {
	var charges []*accessorialcharge.AccessorialCharge
	if err := db.NewSelect().Model(&charges).
		Where("organization_id = ?", refs.orgID).
		Where("status = ?", domaintypes.StatusActive).
		Scan(ctx); err != nil {
		return fmt.Errorf("load accessorial charges: %w", err)
	}
	for _, c := range charges {
		refs.accessorials[strings.ToUpper(c.Code)] = c
	}

	return nil
}

func loadTemplates(ctx context.Context, db bun.IDB, refs *tenantRefs) error {
	if err := db.NewSelect().Model(&refs.templates).
		Where("organization_id = ?", refs.orgID).
		Where("status = ?", formulatemplate.StatusActive).
		Scan(ctx); err != nil {
		return fmt.Errorf("load formula templates: %w", err)
	}
	if len(refs.templates) == 0 {
		return fmt.Errorf("%w: %w", ErrMissingSeedData, errNoFreightTemplates)
	}

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
