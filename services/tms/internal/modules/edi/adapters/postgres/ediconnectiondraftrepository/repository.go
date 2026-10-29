package ediconnectiondraftrepository

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument/edidocumentdb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject/ladingprojectdb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure/servicefailuredb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity/commoditydb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype/servicetypedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype/shipmenttypedb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const configPartnerAS2ID = "partnerAS2Id"

var partnerErrors = dberror.Mapping{Entity: "EDIPartner"}

var identifyingStatuses = []ladingproject.LadingVersionStatus{
	ladingproject.LadingVersionStatusDraft,
	ladingproject.LadingVersionStatusCertified,
	ladingproject.LadingVersionStatusActive,
	ladingproject.LadingVersionStatusShadow,
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) ediconnectiondraftports.EDIConnectionDraftRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-connection-draft-repository"),
	}
}

type matchSet struct {
	matches map[pulid.ID]*ediconnectiondraftports.EDIPartnerMatch
	order   []pulid.ID
}

func newMatchSet() *matchSet {
	return &matchSet{
		matches: make(map[pulid.ID]*ediconnectiondraftports.EDIPartnerMatch),
		order:   make([]pulid.ID, 0, 4),
	}
}

func (m *matchSet) add(partner *edi.EDIPartner) *ediconnectiondraftports.EDIPartnerMatch {
	match, ok := m.matches[partner.ID]
	if !ok {
		match = &ediconnectiondraftports.EDIPartnerMatch{Partner: partner}
		m.matches[partner.ID] = match
		m.order = append(m.order, partner.ID)
	}
	return match
}

func (m *matchSet) missing(ids ...[]pulid.ID) []pulid.ID {
	out := make([]pulid.ID, 0)
	for _, group := range ids {
		for _, id := range group {
			if _, ok := m.matches[id]; !ok && !slices.Contains(out, id) {
				out = append(out, id)
			}
		}
	}
	return out
}

func (m *matchSet) mark(ids []pulid.ID, apply func(*ediconnectiondraftports.EDIPartnerMatch)) {
	for _, id := range ids {
		if match, ok := m.matches[id]; ok {
			apply(match)
		}
	}
}

func (m *matchSet) list() []*ediconnectiondraftports.EDIPartnerMatch {
	out := make([]*ediconnectiondraftports.EDIPartnerMatch, 0, len(m.order))
	for _, id := range m.order {
		out = append(out, m.matches[id])
	}
	return out
}

func (r *repository) FindPartnerMatches(
	ctx context.Context,
	req *ediconnectiondraftports.FindEDIPartnerMatchesRequest,
) ([]*ediconnectiondraftports.EDIPartnerMatch, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*ediconnectiondraftports.EDIPartnerMatch, error) {
			set := newMatchSet()
			direct, err := r.partnersByIdentity(ctx, req)
			if err != nil {
				return nil, err
			}
			code := strings.TrimSpace(req.Code)
			name := strings.TrimSpace(req.Name)
			for _, partner := range direct {
				match := set.add(partner)
				match.ByCode = code != "" && strings.EqualFold(partner.Code, code)
				match.ByName = name != "" && strings.EqualFold(partner.Name, name)
				match.ByCustomer = req.CustomerID.IsNotNil() && partner.CustomerID == req.CustomerID
			}

			byInterchange, err := r.partnerIDsByInterchange(ctx, req)
			if err != nil {
				return nil, err
			}
			byAS2, err := r.partnerIDsByAS2(ctx, req)
			if err != nil {
				return nil, err
			}
			if extra := set.missing(byInterchange, byAS2); len(extra) > 0 {
				partners, loadErr := r.partnersByIDs(ctx, extra)
				if loadErr != nil {
					return nil, loadErr
				}
				for _, partner := range partners {
					set.add(partner)
				}
			}
			set.mark(
				byInterchange,
				func(match *ediconnectiondraftports.EDIPartnerMatch) { match.ByInterchange = true },
			)
			set.mark(
				byAS2,
				func(match *ediconnectiondraftports.EDIPartnerMatch) { match.ByAS2 = true },
			)
			return set.list(), nil
		},
	)
}

func (r *repository) partnersByIdentity(
	ctx context.Context,
	req *ediconnectiondraftports.FindEDIPartnerMatchesRequest,
) ([]*edi.EDIPartner, error) {
	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	if code == "" && name == "" && req.CustomerID.IsNil() {
		return nil, nil
	}

	ep := edidb.EDIPartners
	return ep.Where(keel.Or(
		keel.When(code != "", ep.Code.Lower().Eq(strings.ToLower(code))),
		keel.When(name != "", ep.Name.Lower().Eq(strings.ToLower(name))),
		keel.When(req.CustomerID.IsNotNil(), ep.CustomerID.Eq(req.CustomerID)),
	)).
		Order(ep.CreatedAt.Asc()).
		All(ctx, r.db.Keel(ctx))
}

func (r *repository) partnerIDsByInterchange(
	ctx context.Context,
	req *ediconnectiondraftports.FindEDIPartnerMatchesRequest,
) ([]pulid.ID, error) {
	isaID := strings.TrimSpace(req.ISAID)
	if isaID == "" {
		return nil, nil
	}

	db := r.db.Keel(ctx)
	lv := ladingprojectdb.LadingVersions
	qualifier := strings.TrimSpace(req.ISAQualifier)
	projectIDs, err := keel.Into[pulid.ID](ctx, db, lv.Pick(lv.ProjectID).
		DistinctOn(lv.ProjectID).
		Where(
			lv.Status.In(identifyingStatuses...),
			lv.Manifest.Text("partner", "interchange", "them", "id").Eq(isaID),
			keel.When(
				qualifier != "",
				lv.Manifest.Text("partner", "interchange", "them", "qualifier").Eq(qualifier),
			),
		))
	if err != nil || len(projectIDs) == 0 {
		return nil, err
	}
	lp := ladingprojectdb.LadingProjects
	return keel.Into[pulid.ID](ctx, db, lp.Pick(lp.EDIPartnerID).
		Where(lp.ID.In(projectIDs...)).
		Limit(len(projectIDs)))
}

func (r *repository) partnerIDsByAS2(
	ctx context.Context,
	req *ediconnectiondraftports.FindEDIPartnerMatchesRequest,
) ([]pulid.ID, error) {
	as2ID := strings.TrimSpace(req.AS2ID)
	if as2ID == "" {
		return nil, nil
	}

	cp := edidb.EDICommunicationProfiles
	return keel.Into[pulid.ID](ctx, r.db.Keel(ctx), cp.Pick(cp.EDIPartnerID).
		DistinctOn(cp.EDIPartnerID).
		Where(
			cp.Method.Eq(edi.ConnectionMethodAS2),
			cp.EDIPartnerID.IsNotNull(),
			cp.Config.Text(configPartnerAS2ID).Eq(as2ID),
		))
}

func (r *repository) partnersByIDs(ctx context.Context, ids []pulid.ID) ([]*edi.EDIPartner, error) {
	ep := edidb.EDIPartners
	return ep.Where(ep.ID.In(ids...)).
		Order(ep.CreatedAt.Asc()).
		Limit(len(ids)).
		List(ctx, r.db.Keel(ctx))
}

func (r *repository) GetPartnerSetup(
	ctx context.Context,
	req *ediconnectiondraftports.GetEDIPartnerSetupRequest,
) (*ediconnectiondraftports.EDIPartnerSetup, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*ediconnectiondraftports.EDIPartnerSetup, error) {
			db := r.db.Keel(ctx)
			ep := edidb.EDIPartners
			cp := edidb.EDICommunicationProfiles
			lp := ladingprojectdb.LadingProjects
			mi := edidb.EDIMappingProfileItems
			cn := edidocumentdb.EDIControlNumberSequences

			setup := new(ediconnectiondraftports.EDIPartnerSetup)
			var projects []*ladingproject.LadingProject
			err := keel.Batch(
				ctx,
				db,
				keel.OneInto(ep.Where(ep.ID.Eq(req.PartnerID)), &setup.Partner),
				keel.AllInto(
					cp.Where(cp.EDIPartnerID.Eq(req.PartnerID)).Order(cp.CreatedAt.Asc()),
					&setup.CommunicationProfiles,
				),
				keel.AllInto(lp.Where(lp.EDIPartnerID.Eq(req.PartnerID)).Limit(1), &projects),
				keel.AllInto(
					mi.Where(mi.EDIPartnerID.Eq(req.PartnerID)).
						Order(mi.EntityType.Asc(), mi.SourceID.Asc()),
					&setup.MappingItems,
				),
				keel.AllInto(cn.Where(cn.EDIPartnerID.Eq(req.PartnerID)), &setup.ControlNumbers),
			)
			if err != nil {
				return nil, partnerErrors.Map(err, req.PartnerID.String())
			}
			if len(projects) == 0 {
				return setup, nil
			}
			setup.Project = projects[0]
			lv := ladingprojectdb.LadingVersions
			versions, err := lv.Where(lv.ProjectID.Eq(setup.Project.ID)).
				Order(lv.Number.Desc()).
				Limit(1).
				All(ctx, db)
			if err != nil {
				return nil, err
			}
			if len(versions) > 0 {
				setup.LatestVersion = versions[0]
			}
			return setup, nil
		},
	)
}

func (r *repository) GetInterchangeIdentity(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*ediconnectiondraftports.InterchangeIdentity, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*ediconnectiondraftports.InterchangeIdentity, error) {
			lv := ladingprojectdb.LadingVersions
			qualifier := lv.Manifest.Text("partner", "interchange", "us", "qualifier").Coalesce("")
			senderID := lv.Manifest.Text("partner", "interchange", "us", "id")
			applicationID := lv.Manifest.Text("partner", "interchange", "group", "ours").
				Coalesce("")

			identity, err := keel.IntoOne[ediconnectiondraftports.InterchangeIdentity](
				ctx,
				r.db.Keel(ctx),
				lv.
					Where(
						lv.Status.Eq(ladingproject.LadingVersionStatusActive),
						senderID.Trim().NullIf("").IsNotNull(),
					).
					Group(qualifier, senderID, applicationID).
					Pick(qualifier, senderID, applicationID, keel.Count()).
					Order(keel.Count().Desc(), senderID.Asc()).
					Limit(1),
			)
			if err != nil {
				if dberror.IsNotFoundError(err) {
					return &ediconnectiondraftports.InterchangeIdentity{}, nil
				}
				return nil, err
			}

			return &identity, nil
		},
	)
}

type mappingTarget func(ids []pulid.ID) keel.Projection

var mappingTargets = map[edi.MappingEntityType]mappingTarget{
	edi.MappingEntityTypeCustomer: func(ids []pulid.ID) keel.Projection {
		c := customerdb.Customers
		return c.Pick(c.ID, c.Name).Where(c.ID.In(ids...))
	},
	edi.MappingEntityTypeServiceType: func(ids []pulid.ID) keel.Projection {
		st := servicetypedb.ServiceTypes
		return st.Pick(st.ID, st.Code).Where(st.ID.In(ids...))
	},
	edi.MappingEntityTypeShipmentType: func(ids []pulid.ID) keel.Projection {
		st := shipmenttypedb.ShipmentTypes
		return st.Pick(st.ID, st.Code).Where(st.ID.In(ids...))
	},
	edi.MappingEntityTypeFormulaTemplate: func(ids []pulid.ID) keel.Projection {
		ft := formulatemplatedb.FormulaTemplates
		return ft.Pick(ft.ID, ft.Name).Where(ft.ID.In(ids...))
	},
	edi.MappingEntityTypeLocation: func(ids []pulid.ID) keel.Projection {
		l := locationdb.Locations
		return l.Pick(l.ID, l.Name).Where(l.ID.In(ids...))
	},
	edi.MappingEntityTypeCommodity: func(ids []pulid.ID) keel.Projection {
		c := commoditydb.Commodities
		return c.Pick(c.ID, c.Name).Where(c.ID.In(ids...))
	},
	edi.MappingEntityTypeAccessorialCharge: func(ids []pulid.ID) keel.Projection {
		ac := accessorialchargedb.AccessorialCharges
		return ac.Pick(ac.ID, ac.Code).Where(ac.ID.In(ids...))
	},
	edi.MappingEntityTypeServiceFailureReasonCode: func(ids []pulid.ID) keel.Projection {
		rc := servicefailuredb.ReasonCodes
		return rc.Pick(rc.ID, rc.Code).Where(rc.ID.In(ids...))
	},
}

type targetRow struct {
	ID    pulid.ID
	Label string
}

func (r *repository) ResolveMappingTargets(
	ctx context.Context,
	req *ediconnectiondraftports.ResolveEDIMappingTargetsRequest,
) (map[edi.MappingEntityType]map[pulid.ID]string, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[edi.MappingEntityType]map[pulid.ID]string, error) {
			resolved := make(map[edi.MappingEntityType]map[pulid.ID]string, len(req.Targets))
			for entityType, ids := range req.Targets {
				target, ok := mappingTargets[entityType]
				if !ok || len(ids) == 0 {
					continue
				}
				rows, err := keel.Into[targetRow](ctx, r.db.Keel(ctx), target(ids))
				if err != nil {
					return nil, err
				}
				labels := make(map[pulid.ID]string, len(rows))
				for _, row := range rows {
					labels[row.ID] = row.Label
				}
				resolved[entityType] = labels
			}

			return resolved, nil
		},
	)
}
