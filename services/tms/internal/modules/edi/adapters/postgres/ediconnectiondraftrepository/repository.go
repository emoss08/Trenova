package ediconnectiondraftrepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/accessorialchargecol"
	"github.com/emoss08/trenova/pkg/buncolgen/commoditycol"
	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/buncolgen/edicol"
	"github.com/emoss08/trenova/pkg/buncolgen/formulatemplatecol"
	"github.com/emoss08/trenova/pkg/buncolgen/locationcol"
	"github.com/emoss08/trenova/pkg/buncolgen/servicefailurecol"
	"github.com/emoss08/trenova/pkg/buncolgen/servicetypecol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmenttypecol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	envelopeReceiverID        = "{}->>'interchangeReceiverId'"
	envelopeReceiverQualifier = "{}->>'interchangeReceiverQualifier'"
	envelopeSenderID          = "{}->>'interchangeSenderId'"
	envelopeSenderQualifier   = "{}->>'interchangeSenderQualifier'"
	envelopeApplicationSender = "{}->>'applicationSenderCode'"
	configPartnerAS2ID        = "{}->>'partnerAS2Id'"
)

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

func (r *repository) FindPartnerMatches(
	ctx context.Context,
	req *ediports.FindEDIPartnerMatchesRequest,
) ([]*ediports.EDIPartnerMatch, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ediports.EDIPartnerMatch, error) {
		matches := make(map[pulid.ID]*ediports.EDIPartnerMatch)
		order := make([]pulid.ID, 0, 4)
		mark := func(partner *edi.EDIPartner, apply func(*ediports.EDIPartnerMatch)) {
			match, ok := matches[partner.ID]
			if !ok {
				match = &ediports.EDIPartnerMatch{Partner: partner}
				matches[partner.ID] = match
				order = append(order, partner.ID)
			}
			apply(match)
		}

		direct, err := r.partnersByIdentity(ctx, req)
		if err != nil {
			return nil, err
		}
		code := strings.ToLower(strings.TrimSpace(req.Code))
		name := strings.ToLower(strings.TrimSpace(req.Name))
		for _, partner := range direct {
			mark(partner, func(match *ediports.EDIPartnerMatch) {
				match.ByCode = code != "" && strings.ToLower(partner.Code) == code
				match.ByName = name != "" && strings.ToLower(partner.Name) == name
				match.ByCustomer = req.CustomerID.IsNotNil() && partner.CustomerID == req.CustomerID
			})
		}

		byInterchange, err := r.partnerIDsByInterchange(ctx, req)
		if err != nil {
			return nil, err
		}
		byAS2, err := r.partnerIDsByAS2(ctx, req)
		if err != nil {
			return nil, err
		}
		extra := make([]pulid.ID, 0, len(byInterchange)+len(byAS2))
		for _, id := range append(append([]pulid.ID{}, byInterchange...), byAS2...) {
			if _, ok := matches[id]; !ok {
				extra = append(extra, id)
			}
		}
		if len(extra) > 0 {
			partners, loadErr := r.partnersByIDs(ctx, req.TenantInfo, extra)
			if loadErr != nil {
				return nil, loadErr
			}
			for _, partner := range partners {
				mark(partner, func(*ediports.EDIPartnerMatch) {})
			}
		}
		for _, id := range byInterchange {
			if match, ok := matches[id]; ok {
				match.ByInterchange = true
			}
		}
		for _, id := range byAS2 {
			if match, ok := matches[id]; ok {
				match.ByAS2 = true
			}
		}

		out := make([]*ediports.EDIPartnerMatch, 0, len(order))
		for _, id := range order {
			out = append(out, matches[id])
		}
		return out, nil
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

	cols := edicol.EDIPartnerColumns
	partners := make([]*edi.EDIPartner, 0, 2)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&partners).
		Apply(edicol.EDIPartnerApplyTenant(req.TenantInfo)).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			if code != "" {
				q = q.WhereOr(cols.Code.Expr("LOWER({}) = LOWER(?)"), code)
			}
			if name != "" {
				q = q.WhereOr(cols.Name.Expr("LOWER({}) = LOWER(?)"), name)
			}
			if req.CustomerID.IsNotNil() {
				q = q.WhereOr(cols.CustomerID.Eq(), req.CustomerID)
			}
			return q
		}).
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx)

	return partners, err
}

func (r *repository) partnerIDsByInterchange(
	ctx context.Context,
	req *ediports.FindEDIPartnerMatchesRequest,
) ([]pulid.ID, error) {
	isaID := strings.TrimSpace(req.ISAID)
	if isaID == "" {
		return nil, nil
	}

	cols := edicol.EDIPartnerDocumentProfileColumns
	qualifier := strings.TrimSpace(req.ISAQualifier)
	ids := make([]pulid.ID, 0, 2)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*edi.EDIPartnerDocumentProfile)(nil)).
		Apply(edicol.EDIPartnerDocumentProfileApplyTenant(req.TenantInfo)).
		ColumnExpr(cols.EDIPartnerID.Qualified()).
		Distinct().
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.
				WhereGroup(" OR ", func(q *bun.SelectQuery) *bun.SelectQuery {
					q = q.Where(cols.Direction.Eq(), edi.DocumentDirectionOutbound).
						Where(cols.Envelope.Expr(envelopeReceiverID)+" = ?", isaID)
					if qualifier != "" {
						q = q.Where(cols.Envelope.Expr(envelopeReceiverQualifier)+" = ?", qualifier)
					}
					return q
				}).
				WhereGroup(" OR ", func(q *bun.SelectQuery) *bun.SelectQuery {
					q = q.Where(cols.Direction.Eq(), edi.DocumentDirectionInbound).
						Where(cols.Envelope.Expr(envelopeSenderID)+" = ?", isaID)
					if qualifier != "" {
						q = q.Where(cols.Envelope.Expr(envelopeSenderQualifier)+" = ?", qualifier)
					}
					return q
				})
		}).
		Scan(ctx, &ids)

	return ids, err
}

func (r *repository) partnerIDsByAS2(
	ctx context.Context,
	req *ediports.FindEDIPartnerMatchesRequest,
) ([]pulid.ID, error) {
	as2ID := strings.TrimSpace(req.AS2ID)
	if as2ID == "" {
		return nil, nil
	}

	cols := edicol.EDICommunicationProfileColumns
	ids := make([]pulid.ID, 0, 1)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*edi.EDICommunicationProfile)(nil)).
		Apply(edicol.EDICommunicationProfileApplyTenant(req.TenantInfo)).
		ColumnExpr(cols.EDIPartnerID.Qualified()).
		Distinct().
		Where(cols.Method.Eq(), edi.ConnectionMethodAS2).
		Where(cols.EDIPartnerID.IsNotNull()).
		Where(cols.Config.Expr(configPartnerAS2ID)+" = ?", as2ID).
		Scan(ctx, &ids)

	return ids, err
}

func (r *repository) partnersByIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) ([]*edi.EDIPartner, error) {
	cols := edicol.EDIPartnerColumns
	partners := make([]*edi.EDIPartner, 0, len(ids))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&partners).
		Apply(edicol.EDIPartnerApplyTenant(tenantInfo)).
		Where(cols.ID.In(), bun.List(ids)).
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx)

	return partners, err
}

func (r *repository) GetPartnerSetup(
	ctx context.Context,
	req *ediports.GetEDIPartnerSetupRequest,
) (*ediports.EDIPartnerSetup, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ediports.EDIPartnerSetup, error) {
		db := r.db.DBForContext(ctx)
		setup := &ediports.EDIPartnerSetup{Partner: new(edi.EDIPartner)}

		if err := db.NewSelect().
			Model(setup.Partner).
			Apply(edicol.EDIPartnerApplyTenant(req.TenantInfo)).
			Where(edicol.EDIPartnerColumns.ID.Eq(), req.PartnerID).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, "EDIPartner")
		}

		profileCols := edicol.EDICommunicationProfileColumns
		if err := db.NewSelect().
			Model(&setup.CommunicationProfiles).
			Apply(edicol.EDICommunicationProfileApplyTenant(req.TenantInfo)).
			Where(profileCols.EDIPartnerID.Eq(), req.PartnerID).
			Order(profileCols.CreatedAt.OrderAsc()).
			Scan(ctx); err != nil {
			return nil, err
		}

		documentCols := edicol.EDIPartnerDocumentProfileColumns
		if err := db.NewSelect().
			Model(&setup.DocumentProfiles).
			Apply(edicol.EDIPartnerDocumentProfileApplyTenant(req.TenantInfo)).
			Relation(edicol.EDIPartnerDocumentProfileRelations.Template).
			Where(documentCols.EDIPartnerID.Eq(), req.PartnerID).
			Order(documentCols.CreatedAt.OrderAsc()).
			Scan(ctx); err != nil {
			return nil, err
		}

		itemCols := edicol.EDIMappingProfileItemColumns
		if err := db.NewSelect().
			Model(&setup.MappingItems).
			Apply(edicol.EDIMappingProfileItemApplyTenant(req.TenantInfo)).
			Where(itemCols.EDIPartnerID.Eq(), req.PartnerID).
			Order(itemCols.EntityType.OrderAsc(), itemCols.SourceID.OrderAsc()).
			Scan(ctx); err != nil {
			return nil, err
		}

		sequenceCols := edicol.EDIControlNumberSequenceColumns
		if err := db.NewSelect().
			Model(&setup.ControlNumbers).
			Apply(edicol.EDIControlNumberSequenceApplyTenant(req.TenantInfo)).
			Where(sequenceCols.EDIPartnerID.Eq(), req.PartnerID).
			Scan(ctx); err != nil {
			return nil, err
		}

		return setup, nil
	})
}

func (r *repository) ListTemplateCandidates(
	ctx context.Context,
	req *ediports.ListEDITemplateCandidatesRequest,
) ([]*edi.EDITemplateVersion, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.EDITemplateVersion, error) {
		db := r.db.DBForContext(ctx)
		templateCols := edicol.EDITemplateColumns
		templates := make([]*edi.EDITemplate, 0, 4)
		if err := db.NewSelect().
			Model(&templates).
			Apply(edicol.EDITemplateApplyTenant(req.TenantInfo)).
			Where(templateCols.Standard.Eq(), edi.EDIStandardX12).
			Where(templateCols.TransactionSet.Eq(), req.TransactionSet).
			Where(templateCols.Direction.Eq(), req.Direction).
			Where(templateCols.Status.NotIn(), bun.List([]edi.TemplateStatus{
				edi.TemplateStatusArchived,
				edi.TemplateStatusSuperseded,
			})).
			Order(templateCols.CreatedAt.OrderAsc()).
			Scan(ctx); err != nil {
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

		versionCols := edicol.EDITemplateVersionColumns
		versions := make([]*edi.EDITemplateVersion, 0, len(templates))
		if err := db.NewSelect().
			Model(&versions).
			Apply(edicol.EDITemplateVersionApplyTenant(req.TenantInfo)).
			Relation(edicol.EDITemplateVersionRelations.Segments).
			Where(versionCols.TemplateID.In(), bun.List(ids)).
			Where(versionCols.Status.In(), bun.List([]edi.TemplateStatus{
				edi.TemplateStatusActive,
				edi.TemplateStatusCertified,
			})).
			Order(versionCols.TemplateID.OrderAsc(), versionCols.VersionNumber.OrderDesc()).
			Scan(ctx); err != nil {
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
	tenantInfo pagination.TenantInfo,
) (*ediports.InterchangeIdentity, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ediports.InterchangeIdentity, error) {
		cols := edicol.EDIPartnerDocumentProfileColumns
		identity := new(ediports.InterchangeIdentity)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*edi.EDIPartnerDocumentProfile)(nil)).
			Apply(edicol.EDIPartnerDocumentProfileApplyTenant(tenantInfo)).
			ColumnExpr(cols.Envelope.Expr(envelopeSenderQualifier)+" AS qualifier").
			ColumnExpr(cols.Envelope.Expr(envelopeSenderID)+" AS id").
			ColumnExpr(cols.Envelope.Expr(envelopeApplicationSender)+" AS application_id").
			ColumnExpr("COUNT(*) AS profiles").
			Where(cols.Direction.Eq(), edi.DocumentDirectionOutbound).
			Where(cols.Envelope.Expr("NULLIF(BTRIM("+envelopeSenderID+"), '') IS NOT NULL")).
			GroupExpr("1, 2, 3").
			OrderExpr("profiles DESC, 2 ASC").
			Limit(1).
			Scan(ctx, identity)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return &ediports.InterchangeIdentity{}, nil
			}
			return nil, err
		}
		return identity, nil
	})
}

type mappingTarget struct {
	table buncolgen.TableInfo
	id    buncolgen.Column
	label buncolgen.Column
	org   buncolgen.Column
	bu    buncolgen.Column
}

var mappingTargets = map[edi.MappingEntityType]mappingTarget{
	edi.MappingEntityTypeCustomer: {
		table: customercol.CustomerTable,
		id:    customercol.CustomerColumns.ID,
		label: customercol.CustomerColumns.Name,
		org:   customercol.CustomerColumns.OrganizationID,
		bu:    customercol.CustomerColumns.BusinessUnitID,
	},
	edi.MappingEntityTypeServiceType: {
		table: servicetypecol.ServiceTypeTable,
		id:    servicetypecol.ServiceTypeColumns.ID,
		label: servicetypecol.ServiceTypeColumns.Code,
		org:   servicetypecol.ServiceTypeColumns.OrganizationID,
		bu:    servicetypecol.ServiceTypeColumns.BusinessUnitID,
	},
	edi.MappingEntityTypeShipmentType: {
		table: shipmenttypecol.ShipmentTypeTable,
		id:    shipmenttypecol.ShipmentTypeColumns.ID,
		label: shipmenttypecol.ShipmentTypeColumns.Code,
		org:   shipmenttypecol.ShipmentTypeColumns.OrganizationID,
		bu:    shipmenttypecol.ShipmentTypeColumns.BusinessUnitID,
	},
	edi.MappingEntityTypeFormulaTemplate: {
		table: formulatemplatecol.FormulaTemplateTable,
		id:    formulatemplatecol.FormulaTemplateColumns.ID,
		label: formulatemplatecol.FormulaTemplateColumns.Name,
		org:   formulatemplatecol.FormulaTemplateColumns.OrganizationID,
		bu:    formulatemplatecol.FormulaTemplateColumns.BusinessUnitID,
	},
	edi.MappingEntityTypeLocation: {
		table: locationcol.LocationTable,
		id:    locationcol.LocationColumns.ID,
		label: locationcol.LocationColumns.Name,
		org:   locationcol.LocationColumns.OrganizationID,
		bu:    locationcol.LocationColumns.BusinessUnitID,
	},
	edi.MappingEntityTypeCommodity: {
		table: commoditycol.CommodityTable,
		id:    commoditycol.CommodityColumns.ID,
		label: commoditycol.CommodityColumns.Name,
		org:   commoditycol.CommodityColumns.OrganizationID,
		bu:    commoditycol.CommodityColumns.BusinessUnitID,
	},
	edi.MappingEntityTypeAccessorialCharge: {
		table: accessorialchargecol.AccessorialChargeTable,
		id:    accessorialchargecol.AccessorialChargeColumns.ID,
		label: accessorialchargecol.AccessorialChargeColumns.Code,
		org:   accessorialchargecol.AccessorialChargeColumns.OrganizationID,
		bu:    accessorialchargecol.AccessorialChargeColumns.BusinessUnitID,
	},
	edi.MappingEntityTypeServiceFailureReasonCode: {
		table: servicefailurecol.ReasonCodeTable,
		id:    servicefailurecol.ReasonCodeColumns.ID,
		label: servicefailurecol.ReasonCodeColumns.Code,
		org:   servicefailurecol.ReasonCodeColumns.OrganizationID,
		bu:    servicefailurecol.ReasonCodeColumns.BusinessUnitID,
	},
}

type targetRow struct {
	ID    pulid.ID `bun:"id"`
	Label string   `bun:"label"`
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
				rows := make([]targetRow, 0, len(ids))
				if err := r.db.DBForContext(ctx).
					NewSelect().
					TableExpr(target.table.As(target.table.Alias)).
					ColumnExpr(target.id.As("id")).
					ColumnExpr(target.label.As("label")).
					Where(target.id.In(), bun.List(ids)).
					Where(target.org.Eq(), req.TenantInfo.OrgID).
					Where(target.bu.Eq(), req.TenantInfo.BuID).
					Scan(ctx, &rows); err != nil {
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
