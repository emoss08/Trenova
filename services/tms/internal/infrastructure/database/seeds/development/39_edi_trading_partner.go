package development

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/buncolgen/edicol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/templates"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const (
	SeedEDIPartnerCode  = "ACMEMFG"
	SeedEDIPartnerName  = "Acme Manufacturing EDI"
	seedEDIReceiverID   = "ACMEMFG"
	seedEDISenderID     = "TRENOVA"
	seedEDICarrierSCAC  = "TRNV"
	seedEDIShipmentRows = 9
)

type seedEDIMessage struct {
	set      edi.TransactionSet
	ack      edi.MessageAcknowledgmentStatus
	delivery edi.MessageDeliveryStatus
	age      time.Duration
	status   string
	ackError string
	error    string
}

var seedEDIMessages = []seedEDIMessage{
	{set: edi.TransactionSet214, ack: edi.MessageAcknowledgmentStatusPending,
		delivery: edi.MessageDeliveryStatusSent, age: 31 * time.Hour, status: "X3"},
	{set: edi.TransactionSet214, ack: edi.MessageAcknowledgmentStatusPending,
		delivery: edi.MessageDeliveryStatusSent, age: 19 * time.Hour, status: "AF"},
	{set: edi.TransactionSet214, ack: edi.MessageAcknowledgmentStatusPending,
		delivery: edi.MessageDeliveryStatusSent, age: 6 * time.Hour, status: "X1"},
	{set: edi.TransactionSet214, ack: edi.MessageAcknowledgmentStatusPending,
		delivery: edi.MessageDeliveryStatusSent, age: 40 * time.Minute, status: "AF"},
	{set: edi.TransactionSet214, ack: edi.MessageAcknowledgmentStatusAccepted,
		delivery: edi.MessageDeliveryStatusSent, age: 50 * time.Hour, status: "D1"},
	{set: edi.TransactionSet214, ack: edi.MessageAcknowledgmentStatusAccepted,
		delivery: edi.MessageDeliveryStatusSent, age: 26 * time.Hour, status: "X3"},
	{set: edi.TransactionSet214, ack: edi.MessageAcknowledgmentStatusAcceptedWithErrors,
		delivery: edi.MessageDeliveryStatusSent, age: 22 * time.Hour, status: "X1",
		ackError: "acknowledgment code E; Mandatory data element missing (code 1) segment MS1"},
	{set: edi.TransactionSet214, ack: edi.MessageAcknowledgmentStatusRejected,
		delivery: edi.MessageDeliveryStatusSent, age: 14 * time.Hour, status: "AF",
		ackError: "acknowledgment code R; Invalid code value (code 7) segment AT7"},
	{set: edi.TransactionSet214, ack: edi.MessageAcknowledgmentStatusNotExpected,
		delivery: edi.MessageDeliveryStatusDeadLettered, age: 3 * time.Hour, status: "X6",
		error: "sftp: dial tcp 127.0.0.1:9222: connect: connection refused (5 attempts)"},
}

type EDITradingPartnerSeed struct {
	seedhelpers.BaseSeed
}

func NewEDITradingPartnerSeed() *EDITradingPartnerSeed {
	seed := &EDITradingPartnerSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"EDITradingPartner",
		"1.0.0",
		"Seeds an outside EDI trading partner for Acme Manufacturing with outbound 214 and 990 profiles and 214s in every acknowledgment state",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedTestOrganizations, seedhelpers.SeedShipment)

	return seed
}

func (s *EDITradingPartnerSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return err
			}
			tenant := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}

			exists, err := tx.NewSelect().
				Model((*edi.EDIPartner)(nil)).
				Where(edicol.EDIPartnerColumns.OrganizationID.Eq(), tenant.OrgID).
				Where(edicol.EDIPartnerColumns.BusinessUnitID.Eq(), tenant.BuID).
				Where(edicol.EDIPartnerColumns.Code.Eq(), SeedEDIPartnerCode).
				Exists(ctx)
			if err != nil {
				return fmt.Errorf("look for the seeded EDI partner: %w", err)
			}
			if exists {
				return nil
			}

			partner, err := seedEDIPartner(ctx, tx, tenant)
			if err != nil {
				return err
			}
			profile214, err := seedOutboundProfile(ctx, tx, tenant, partner, edi.TransactionSet214)
			if err != nil {
				return err
			}
			if _, err = seedOutboundProfile(ctx, tx, tenant, partner, edi.TransactionSet990); err != nil {
				return err
			}

			return seedOutboundMessages(ctx, tx, tenant, partner, profile214)
		},
	)
}

func seedEDIPartner(
	ctx context.Context,
	tx bun.Tx,
	tenant pagination.TenantInfo,
) (*edi.EDIPartner, error) {
	acme := new(customer.Customer)
	cols := customercol.CustomerColumns
	err := tx.NewSelect().
		Model(acme).
		Where(cols.OrganizationID.Eq(), tenant.OrgID).
		Where(cols.BusinessUnitID.Eq(), tenant.BuID).
		Where(cols.Code.Eq(), "ACME").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("find the Acme Manufacturing customer: %w", err)
	}

	partner := &edi.EDIPartner{
		BusinessUnitID:     tenant.BuID,
		OrganizationID:     tenant.OrgID,
		Kind:               edi.PartnerKindExternal,
		Status:             domaintypes.StatusActive,
		Code:               SeedEDIPartnerCode,
		Name:               SeedEDIPartnerName,
		Description:        "Acme Manufacturing's EDI desk: tenders in, 214 status and 990 responses out.",
		CustomerID:         acme.ID,
		Timezone:           "America/Chicago",
		Country:            "US",
		ContactName:        "Acme EDI Support",
		ContactEmail:       "edi-support@acme.example",
		EnabledForInbound:  true,
		EnabledForOutbound: true,
		Settings:           map[string]any{},
	}
	if _, err = tx.NewInsert().Model(partner).Exec(ctx); err != nil {
		return nil, fmt.Errorf("insert the EDI partner: %w", err)
	}

	return partner, nil
}

func seedOutboundProfile(
	ctx context.Context,
	tx bun.Tx,
	tenant pagination.TenantInfo,
	partner *edi.EDIPartner,
	set edi.TransactionSet,
) (*edi.EDIPartnerDocumentProfile, error) {
	template, version, err := seedBaseTemplate(ctx, tx, tenant, set)
	if err != nil {
		return nil, err
	}

	envelope := edi.DefaultX12EnvelopeSettings()
	envelope.InterchangeSenderID = seedEDISenderID
	envelope.InterchangeReceiverID = seedEDIReceiverID
	envelope.ApplicationSenderCode = seedEDISenderID
	envelope.ApplicationReceiverCode = seedEDIReceiverID
	acknowledgment := edi.AcknowledgmentConfig{
		Expected:           true,
		Type:               edi.AcknowledgmentType997,
		SLAInMinutes:       240,
		MissingAckSeverity: edi.ValidationSeverityWarning,
	}
	if set == edi.TransactionSet990 {
		acknowledgment = edi.AcknowledgmentConfig{Type: edi.AcknowledgmentTypeNone}
	}

	profile := &edi.EDIPartnerDocumentProfile{
		BusinessUnitID:    tenant.BuID,
		OrganizationID:    tenant.OrgID,
		EDIPartnerID:      partner.ID,
		DocumentTypeID:    template.DocumentTypeID,
		TemplateID:        template.ID,
		TemplateVersionID: version.ID,
		Name:              fmt.Sprintf("%s %s outbound", SeedEDIPartnerCode, set),
		Status:            edi.DocumentStatusActive,
		Direction:         edi.DocumentDirectionOutbound,
		Standard:          edi.EDIStandardX12,
		TransactionSet:    set,
		FunctionalGroupID: edi.FunctionalGroupDefault(set),
		Envelope:          envelope,
		Acknowledgment:    acknowledgment,
		ValidationMode:    edi.ValidationModeStrict,
		PartnerSettings:   map[string]any{"carrier": map[string]any{"scac": seedEDICarrierSCAC}},
	}
	if _, err = tx.NewInsert().Model(profile).Exec(ctx); err != nil {
		return nil, fmt.Errorf("insert the %s document profile: %w", set, err)
	}

	return profile, nil
}

func seedBaseTemplate(
	ctx context.Context,
	tx bun.Tx,
	tenant pagination.TenantInfo,
	set edi.TransactionSet,
) (*edi.EDITemplate, *edi.EDITemplateVersion, error) {
	documentType := new(edi.EDIDocumentType)
	typeCols := edicol.EDIDocumentTypeColumns
	err := tx.NewSelect().
		Model(documentType).
		Where(typeCols.Standard.Eq(), edi.EDIStandardX12).
		Where(typeCols.TransactionSet.Eq(), set).
		Where(typeCols.Direction.Eq(), edi.DocumentDirectionOutbound).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("find the X12 %s outbound document type: %w", set, err)
	}

	name := fmt.Sprintf("Base X12 %s %s", set, edi.DocumentDirectionOutbound)
	existing, existingVersion, err := findBaseTemplate(ctx, tx, tenant, set, name)
	if err != nil {
		return nil, nil, err
	}
	if existing != nil {
		return existing, existingVersion, nil
	}

	template := &edi.EDITemplate{
		BusinessUnitID: tenant.BuID,
		OrganizationID: tenant.OrgID,
		DocumentTypeID: documentType.ID,
		Name:           name,
		Description:    fmt.Sprintf("Tenant-scoped base outbound X12 %s template", set),
		Direction:      edi.DocumentDirectionOutbound,
		Standard:       edi.EDIStandardX12,
		TransactionSet: set,
		Status:         edi.TemplateStatusActive,
	}
	if _, err = tx.NewInsert().Model(template).Exec(ctx); err != nil {
		return nil, nil, fmt.Errorf("insert the base %s template: %w", set, err)
	}

	activatedAt := timeutils.NowUnix()
	version := &edi.EDITemplateVersion{
		BusinessUnitID:    tenant.BuID,
		OrganizationID:    tenant.OrgID,
		TemplateID:        template.ID,
		VersionNumber:     1,
		X12Version:        edi.DefaultX12204Version,
		FunctionalGroupID: edi.FunctionalGroupDefault(set),
		Status:            edi.TemplateStatusActive,
		IsActive:          true,
		Notes:             fmt.Sprintf("Seeded base %s X12 %s outbound profile", edi.DefaultX12204Version, set),
		ActivatedAt:       &activatedAt,
	}
	if _, err = tx.NewInsert().Model(version).Exec(ctx); err != nil {
		return nil, nil, fmt.Errorf("insert the base %s template version: %w", set, err)
	}

	segments, err := templates.StarterSegments(tenant, version.ID, set)
	if err != nil {
		return nil, nil, err
	}
	if _, err = tx.NewInsert().Model(&segments).Exec(ctx); err != nil {
		return nil, nil, fmt.Errorf("insert the base %s template segments: %w", set, err)
	}

	return template, version, nil
}

func findBaseTemplate(
	ctx context.Context,
	tx bun.Tx,
	tenant pagination.TenantInfo,
	set edi.TransactionSet,
	name string,
) (*edi.EDITemplate, *edi.EDITemplateVersion, error) {
	template := new(edi.EDITemplate)
	cols := edicol.EDITemplateColumns
	err := tx.NewSelect().
		Model(template).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return edicol.EDITemplateScopeTenant(sq, tenant).
				Where(cols.Standard.Eq(), edi.EDIStandardX12).
				Where(cols.TransactionSet.Eq(), set).
				Where(cols.Direction.Eq(), edi.DocumentDirectionOutbound).
				Where(cols.Name.Eq(), name)
		}).
		Limit(1).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("look for the base %s template: %w", set, err)
	}

	version := new(edi.EDITemplateVersion)
	versionCols := edicol.EDITemplateVersionColumns
	if err = tx.NewSelect().
		Model(version).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return edicol.EDITemplateVersionScopeTenant(sq, tenant).
				Where(versionCols.TemplateID.Eq(), template.ID).
				Where(versionCols.IsActive.IsTrue())
		}).
		Limit(1).
		Scan(ctx); err != nil {
		return nil, nil, fmt.Errorf("find the active version of the base %s template: %w", set, err)
	}

	return template, version, nil
}

func seedOutboundMessages(
	ctx context.Context,
	tx bun.Tx,
	tenant pagination.TenantInfo,
	partner *edi.EDIPartner,
	profile *edi.EDIPartnerDocumentProfile,
) error {
	shipments := make([]*shipment.Shipment, 0, seedEDIShipmentRows)
	cols := shipmentcol.ShipmentColumns
	if err := tx.NewSelect().
		Model(&shipments).
		Where(cols.OrganizationID.Eq(), tenant.OrgID).
		Where(cols.BusinessUnitID.Eq(), tenant.BuID).
		Where(cols.CustomerID.Eq(), partner.CustomerID).
		Order(cols.ProNumber.OrderAsc()).
		Limit(seedEDIShipmentRows).
		Scan(ctx); err != nil {
		return fmt.Errorf("read Acme Manufacturing's shipments: %w", err)
	}
	if len(shipments) == 0 {
		return errors.New("the development seed has no Acme Manufacturing shipment to report on")
	}

	now := time.Now().UTC()
	for idx, entry := range seedEDIMessages {
		load := shipments[idx%len(shipments)]
		generated := now.Add(-entry.age)
		control := idx + 1
		message := &edi.EDIMessage{
			BusinessUnitID:           tenant.BuID,
			OrganizationID:           tenant.OrgID,
			EDIPartnerID:             partner.ID,
			DocumentTypeID:           profile.DocumentTypeID,
			PartnerDocumentProfileID: profile.ID,
			TemplateID:               profile.TemplateID,
			TemplateVersionID:        profile.TemplateVersionID,
			ShipmentID:               load.ID,
			Direction:                edi.DocumentDirectionOutbound,
			Standard:                 edi.EDIStandardX12,
			TransactionSet:           entry.set,
			X12Version:               edi.DefaultX12204Version,
			Status:                   edi.MessageStatusGenerated,
			ValidationMode:           edi.ValidationModeStrict,
			InterchangeControlNumber: fmt.Sprintf("%09d", 5000+control),
			GroupControlNumber:       fmt.Sprintf("%d", 5000+control),
			TransactionControlNumber: fmt.Sprintf("%04d", control),
			SegmentCount:             5,
			RawX12:                   seed214(generated, control, load, entry.status),
			PayloadSnapshot: edi.DocumentPayload{
				TransactionSet: edi.TransactionSet214,
				ShipmentStatus: &edi.ShipmentStatusPayload{
					ShipmentID: load.ID,
					BOL:        load.BOL,
					ProNumber:  load.ProNumber,
					StatusCode: entry.status,
					EventDate:  generated.Unix(),
					EventTime:  generated.Unix(),
				},
			},
			DeliveryStatus:    entry.delivery,
			DeliveryAttempts:  1,
			DeliveryLastError: entry.error,
			AckStatus:         entry.ack,
			AckLastError:      entry.ackError,
			GeneratedAt:       generated.Unix(),
		}
		if entry.delivery == edi.MessageDeliveryStatusDeadLettered {
			message.DeliveryAttempts = 5
		}
		if entry.delivery == edi.MessageDeliveryStatusSent {
			sentAt := generated.Add(30 * time.Second).Unix()
			message.DeliverySentAt = &sentAt
		}
		if entry.ack != edi.MessageAcknowledgmentStatusPending &&
			entry.ack != edi.MessageAcknowledgmentStatusNotExpected {
			ackedAt := generated.Add(9 * time.Minute).Unix()
			message.AckReceivedAt = &ackedAt
		}
		if _, err := tx.NewInsert().Model(message).Exec(ctx); err != nil {
			return fmt.Errorf("insert 214 %d: %w", control, err)
		}
		if _, err := tx.NewUpdate().
			Model((*edi.EDIMessage)(nil)).
			Set(edicol.EDIMessageColumns.CreatedAt.Set(), generated.Unix()).
			Set(edicol.EDIMessageColumns.UpdatedAt.Set(), generated.Unix()).
			Where(edicol.EDIMessageColumns.ID.Eq(), message.ID).
			Where(edicol.EDIMessageColumns.OrganizationID.Eq(), tenant.OrgID).
			Where(edicol.EDIMessageColumns.BusinessUnitID.Eq(), tenant.BuID).
			Exec(ctx); err != nil {
			return fmt.Errorf("date 214 %d: %w", control, err)
		}
	}

	return nil
}

func seed214(at time.Time, control int, load *shipment.Shipment, status string) string {
	interchange := fmt.Sprintf("%09d", 5000+control)
	group := fmt.Sprintf("%d", 5000+control)
	set := fmt.Sprintf("%04d", control)
	segments := []string{
		fmt.Sprintf("ISA*00*          *00*          *ZZ*%-15s*ZZ*%-15s*%s*%s*^*00401*%s*0*P*>",
			seedEDISenderID, seedEDIReceiverID, at.Format("060102"), at.Format("1504"), interchange),
		fmt.Sprintf("GS*QM*%s*%s*%s*%s*%s*X*004010",
			seedEDISenderID, seedEDIReceiverID, at.Format("20060102"), at.Format("1504"), group),
		"ST*214*" + set,
		fmt.Sprintf("B10*%s*%s*%s", load.ProNumber, load.BOL, seedEDICarrierSCAC),
		"LX*1",
		fmt.Sprintf("AT7*%s*NS***%s*%s", status, at.Format("20060102"), at.Format("1504")),
		"SE*5*" + set,
		"GE*1*" + group,
		"IEA*1*" + interchange,
	}

	return strings.Join(segments, "~") + "~"
}
