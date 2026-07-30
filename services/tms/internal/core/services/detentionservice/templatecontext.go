package detentionservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

// ContextBuilder turns a detention occurrence into the variables its notice
// templates declare.
//
// Like the invoice builder, it depends on repositories rather than on *Service:
// the detention service renders through the template resolver, and a builder
// routed through the service would close that fx cycle.
type ContextBuilder struct {
	occurrenceRepo repositories.DetentionOccurrenceRepository
	orgRepo        repositories.OrganizationRepository
	inliner        services.AssetInliner
}

type ContextBuilderParams struct {
	fx.In

	OccurrenceRepo repositories.DetentionOccurrenceRepository
	OrgRepo        repositories.OrganizationRepository
	Inliner        services.AssetInliner
}

func NewContextBuilder(p ContextBuilderParams) *ContextBuilder {
	return &ContextBuilder{
		occurrenceRepo: p.OccurrenceRepo,
		orgRepo:        p.OrgRepo,
		inliner:        p.Inliner,
	}
}

var _ services.ContextBuilder = (*ContextBuilder)(nil)

// Kind reports the email notice. The PDF variant shares the same context, and
// the resolver looks up a builder by the kind being rendered, so the PDF kind is
// registered separately by pdfContextBuilder below.
func (b *ContextBuilder) Kind() documenttemplate.Kind {
	return documenttemplate.KindDetentionNoticeEmail
}

func (b *ContextBuilder) Build(
	ctx context.Context,
	req *services.BuildContextRequest,
) (any, error) {
	occurrence, err := b.occurrenceRepo.GetByID(
		ctx,
		&repositories.GetDetentionOccurrenceByIDRequest{
			OccurrenceID: req.ReferenceID,
			TenantInfo:   req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	return b.BuildFrom(ctx, &buildNoticeContextParams{
		Occurrence: occurrence,
		Kind:       detention.NoticeKindFinal,
		TenantInfo: req.TenantInfo,
	})
}

// pdfContextBuilder registers the same context under the PDF kind.
//
// A notice PDF is the email's content on paper — same figures, same wording,
// different channel — so duplicating the builder would create two places for the
// numbers in a dispute to disagree.
type pdfContextBuilder struct{ *ContextBuilder }

func NewPDFContextBuilder(b *ContextBuilder) services.ContextBuilder {
	return pdfContextBuilder{ContextBuilder: b}
}

func (pdfContextBuilder) Kind() documenttemplate.Kind {
	return documenttemplate.KindDetentionNoticePDF
}

// buildNoticeContextParams is everything a notice cites.
//
// Every figure comes from the occurrence's frozen policy snapshot rather than
// live configuration, so the notice and the invoice that follows quote identical
// terms even if someone edits the policy in between.
type buildNoticeContextParams struct {
	Occurrence    *detention.DetentionOccurrence
	Kind          detention.NoticeKind
	FacilityName  string
	ShipmentRef   string
	CustomerName  string
	ArrivalSource detention.EvidenceSource
	TenantInfo    pagination.TenantInfo
	Location      *time.Location
}

func (b *ContextBuilder) BuildFrom(
	ctx context.Context,
	p *buildNoticeContextParams,
) (*documenttemplate.DetentionNoticeContext, error) {
	occ := p.Occurrence
	loc := p.Location
	if loc == nil {
		loc = time.UTC
	}

	facility := p.FacilityName
	if facility == "" {
		facility = "the facility"
	}

	ref := p.ShipmentRef
	if ref == "" {
		ref = occ.ShipmentID.String()
	}

	out := &documenttemplate.DetentionNoticeContext{
		Kind:              string(p.Kind),
		ShipmentProNumber: ref,
		FacilityName:      facility,
		CustomerName:      p.CustomerName,
		Currency:          occ.Currency,
	}

	if occ.ArrivedAt != nil {
		out.ArrivedAt = formatStamp(*occ.ArrivedAt, loc)
	}
	if p.ArrivalSource != "" {
		out.ArrivalSource = string(p.ArrivalSource)
	}
	if occ.DepartedAt != nil {
		out.DepartedAt = formatStamp(*occ.DepartedAt, loc)
	}

	if snap := occ.PolicySnapshot; snap != nil {
		out.FreeTime = formatMinutes(snap.FreeMinutes)
		out.FreeTimeExpiresAt = formatStamp(occ.FreeTimeExpiresAt, loc)
		out.RateLabel = noticeRateLabel(snap)
		out.Tiers = noticeTierRows(snap)
	}

	// A warning fires before anything has accrued, so these stay empty rather
	// than printing a zero a customer would read as a charge.
	if occ.RoundedMinutes > 0 || p.Kind == detention.NoticeKindFinal {
		out.TotalTimeOnSite = formatMinutes(occ.RawDwellMinutes)
		out.BillableTime = formatMinutes(occ.RoundedMinutes)
		out.BillableAmount = occ.BillableAmount.StringFixed(2)
	}

	if err := b.applyBranding(ctx, occ.OrganizationID, occ.BusinessUnitID, out); err != nil {
		return nil, err
	}

	return out, nil
}

// applyBranding fills the company name and logo from the organization.
func (b *ContextBuilder) applyBranding(
	ctx context.Context,
	orgID, buID pulid.ID,
	out *documenttemplate.DetentionNoticeContext,
) error {
	if b.orgRepo == nil {
		return nil
	}

	org, err := b.orgRepo.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	})
	if err != nil || org == nil {
		// A notice that cannot name the sender is still worth sending; the
		// charge behind it does not depend on the letterhead.
		return nil //nolint:nilerr // branding is decoration, not evidence
	}

	out.CompanyName = org.Name

	dataURI, err := services.ResolveLogoDataURI(ctx, b.inliner, org.LogoURL)
	if err != nil {
		return err
	}
	out.LogoDataURI = dataURI

	return nil
}

// noticeTierRows exposes the rate schedule a tiered charge was derived from.
//
// A customer disputing a tiered charge asks which band applied; showing the
// whole schedule answers that without a phone call.
func noticeTierRows(snap *detention.PolicySnapshot) []documenttemplate.DetentionTierRow {
	if snap.RateSource != detention.RateSourceTiers || len(snap.Tiers) == 0 {
		return nil
	}

	rows := make([]documenttemplate.DetentionTierRow, 0, len(snap.Tiers))
	for _, tier := range snap.Tiers {
		rows = append(rows, documenttemplate.DetentionTierRow{
			Band:  tierBandTerm(tier),
			Rate:  rateLabel(tier.Rate, tier.RateUnit),
			Label: fmt.Sprintf("%s at %s", tierBandTerm(tier), rateLabel(tier.Rate, tier.RateUnit)),
		})
	}

	return rows
}
