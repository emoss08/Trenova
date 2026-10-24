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
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
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

const (
	envelopeReceiverID        = "interchangeReceiverId"
	envelopeReceiverQualifier = "interchangeReceiverQualifier"
	envelopeSenderID          = "interchangeSenderId"
	envelopeSenderQualifier   = "interchangeSenderQualifier"
	envelopeApplicationSender = "applicationSenderCode"
	configPartnerAS2ID        = "partnerAS2Id"
)

var partnerErrors = dberror.Mapping{Entity: "EDIPartner"}

var excludedTemplateStatuses = []edi.TemplateStatus{
	edi.TemplateStatusArchived,
	edi.TemplateStatusSuperseded,
}

var candidateVersionStatuses = []edi.TemplateStatus{
	edi.TemplateStatusActive,
	edi.TemplateStatusCertified,
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

func New(p Params) ediports.EDIConnectionDraftRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-connection-draft-repository"),
	}
}

type matchSet struct {
	matches map[pulid.ID]*ediports.EDIPartnerMatch
	order   []pulid.ID
}

func newMatchSet() *matchSet {
	return &matchSet{
		matches: make(map[pulid.ID]*ediports.EDIPartnerMatch),
		order:   make([]pulid.ID, 0, 4),
	}
}

func (m *matchSet) add(partner *edi.EDIPartner) *ediports.EDIPartnerMatch {
	match, ok := m.matches[partner.ID]
	if !ok {
		match = &ediports.EDIPartnerMatch{Partner: partner}
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

func (m *matchSet) mark(ids []pulid.ID, apply func(*ediports.EDIPartnerMatch)) {
	for _, id := range ids {
		if match, ok := m.matches[id]; ok {
			apply(match)
		}
	}
}

func (m *matchSet) list() []*ediports.EDIPartnerMatch {
	out := make([]*ediports.EDIPartnerMatch, 0, len(m.order))
	for _, id := range m.order {
		out = append(out, m.matches[id])
	}
	return out
}

func (r *repository) FindPartnerMatches(
	ctx context.Context,
	req *ediports.FindEDIPartnerMatchesRequest,
) ([]*ediports.EDIPartnerMatch, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ediports.EDIPartnerMatch, error) {
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
			func(match *ediports.EDIPartnerMatch) { match.ByInterchange = true },
		)
		set.mark(byAS2, func(match *ediports.EDIPartnerMatch) { match.ByAS2 = true })
		return set.list(), nil
	})
}

func (r *repository) partnersByIdentity(
	ctx context.Context,
	req *ediports.FindEDIPartnerMatchesRequest,
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
	req *ediports.FindEDIPartnerMatchesRequest,
) ([]pulid.ID, error) {
	isaID := strings.TrimSpace(req.ISAID)
	if isaID == "" {
		return nil, nil
	}

	dp := edidb.EDIPartnerDocumentProfiles
	qualifier := strings.TrimSpace(req.ISAQualifier)
	return keel.Into[pulid.ID](ctx, r.db.Keel(ctx), dp.Pick(dp.EDIPartnerID).
		DistinctOn(dp.EDIPartnerID).
		Where(keel.Or(
			keel.And(
				dp.Direction.Eq(edi.DocumentDirectionOutbound),
				dp.Envelope.Text(envelopeReceiverID).Eq(isaID),
				keel.When(qualifier != "", dp.Envelope.Text(envelopeReceiverQualifier).Eq(qualifier)),
			),
			keel.And(
				dp.Direction.Eq(edi.DocumentDirectionInbound),
				dp.Envelope.Text(envelopeSenderID).Eq(isaID),
				keel.When(qualifier != "", dp.Envelope.Text(envelopeSenderQualifier).Eq(qualifier)),
			),
		)))
}

func (r *repository) partnerIDsByAS2(
	ctx context.Context,
	req *ediports.FindEDIPartnerMatchesRequest,
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
	req *ediports.GetEDIPartnerSetupRequest,
) (*ediports.EDIPartnerSetup, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ediports.EDIPartnerSetup, error) {
		db := r.db.Keel(ctx)
		ep := edidb.EDIPartners
		cp := edidb.EDICommunicationProfiles
		dp := edidb.EDIPartnerDocumentProfiles
		mi := edidb.EDIMappingProfileItems
		cn := edidb.EDIControlNumberSequences

		setup := new(ediports.EDIPartnerSetup)
		err := keel.Batch(
			ctx,
			db,
			keel.OneInto(ep.Where(ep.ID.Eq(req.PartnerID)), &setup.Partner),
			keel.AllInto(
				cp.Where(cp.EDIPartnerID.Eq(req.PartnerID)).Order(cp.CreatedAt.Asc()),
				&setup.CommunicationProfiles,
			),
			keel.AllInto(
				dp.Where(dp.EDIPartnerID.Eq(req.PartnerID)).
					Load(dp.Template()).
					Order(dp.CreatedAt.Asc()),
				&setup.DocumentProfiles,
			),
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

		return setup, nil
	})
}

func (r *repository) ListTemplateCandidates(
	ctx context.Context,
	req *ediports.ListEDITemplateCandidatesRequest,
) ([]*edi.EDITemplateVersion, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.EDITemplateVersion, error) {
		db := r.db.Keel(ctx)
		t := edidb.EDITemplates
		templates, err := t.Where(
			t.Standard.Eq(edi.EDIStandardX12),
			t.TransactionSet.Eq(req.TransactionSet),
			t.Direction.Eq(req.Direction),
			t.Status.NotIn(excludedTemplateStatuses...),
		).
			Order(t.CreatedAt.Asc()).
			All(ctx, db)
		if err != nil {
			return nil, err
		}
		if len(templates) == 0 {
			return []*edi.EDITemplateVersion{}, nil
		}

		byID := make(map[pulid.ID]*edi.EDITemplate, len(templates))
		ids := make([]pulid.ID, 0, len(templates))
		for _, template := range templates {
			byID[template.ID] = template
			ids = append(ids, template.ID)
		}

		tv := edidb.EDITemplateVersions
		versions, err := tv.Where(tv.TemplateID.In(ids...), tv.Status.In(candidateVersionStatuses...)).
			Load(tv.Segments()).
			Order(tv.TemplateID.Asc(), tv.VersionNumber.Desc()).
			All(ctx, db)
		if err != nil {
			return nil, err
		}

		latest := make([]*edi.EDITemplateVersion, 0, len(versions))
		seen := make(map[pulid.ID]struct{}, len(versions))
		for _, version := range versions {
			if _, ok := seen[version.TemplateID]; ok && !version.IsActive {
				continue
			}
			if _, ok := seen[version.TemplateID]; ok {
				latest = replaceVersion(latest, version)
				continue
			}
			seen[version.TemplateID] = struct{}{}
			version.Template = byID[version.TemplateID]
			latest = append(latest, version)
		}

		return latest, nil
	})
}

func replaceVersion(
	versions []*edi.EDITemplateVersion,
	active *edi.EDITemplateVersion,
) []*edi.EDITemplateVersion {
	for index, version := range versions {
		if version.TemplateID == active.TemplateID {
			active.Template = version.Template
			versions[index] = active
		}
	}
	return versions
}

func (r *repository) GetInterchangeIdentity(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*ediports.InterchangeIdentity, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ediports.InterchangeIdentity, error) {
		dp := edidb.EDIPartnerDocumentProfiles
		qualifier := keel.Raw("COALESCE(?, '')", dp.Envelope.Text(envelopeSenderQualifier))
		senderID := dp.Envelope.Text(envelopeSenderID)
		applicationID := keel.Raw("COALESCE(?, '')", dp.Envelope.Text(envelopeApplicationSender))

		identity, err := keel.IntoOne[ediports.InterchangeIdentity](ctx, r.db.Keel(ctx), dp.
			Where(
				dp.Direction.Eq(edi.DocumentDirectionOutbound),
				keel.Raw("NULLIF(BTRIM(?), '') IS NOT NULL", senderID),
			).
			Group(qualifier, senderID, applicationID).
			Pick(qualifier, senderID, applicationID, keel.Count()).
			Order(keel.Count().Desc(), senderID.Asc()).
			Limit(1))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return &ediports.InterchangeIdentity{}, nil
			}
			return nil, err
		}

		return &identity, nil
	})
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
	req *ediports.ResolveEDIMappingTargetsRequest,
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
