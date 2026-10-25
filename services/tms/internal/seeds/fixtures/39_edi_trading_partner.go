package fixtures

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/templates"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
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
}

var EDITradingPartner = Seeds.Add(seed.Seed{
	Name:        "edi_trading_partner",
	Kind:        seed.Fixture,
	Description: "Seeds an outside EDI trading partner for Acme Manufacturing with outbound 214 and 990 profiles and 214s in every acknowledgment state",
	After:       []*seed.Seed{AdminAccount, Shipment},
	Run:         (&EDITradingPartnerSeed{}).Run,
})

func (s *EDITradingPartnerSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}
	tenant := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}

	partner, err := seedEDIPartner(ctx, db, tenant)
	if err != nil {
		return err
	}
	profile214, err := seedOutboundProfile(ctx, db, tenant, partner, edi.TransactionSet214)
	if err != nil {
		return err
	}
	if _, err = seedOutboundProfile(ctx, db, tenant, partner, edi.TransactionSet990); err != nil {
		return err
	}

	return seedOutboundMessages(ctx, db, tenant, partner, profile214)
}

func seedEDIPartner(
	ctx context.Context,
	db *keel.DB,
	tenant pagination.TenantInfo,
) (*edi.EDIPartner, error) {
	acme, err := customerdb.Customers.AllTenants().Where(
		customerdb.Customers.OrganizationID.Eq(tenant.OrgID),
		customerdb.Customers.BusinessUnitID.Eq(tenant.BuID),
		customerdb.Customers.Code.Eq("ACME"),
	).First(ctx, db)
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
	if err = edidb.EDIPartners.Create(ctx, db, partner); err != nil {
		return nil, fmt.Errorf("insert the EDI partner: %w", err)
	}

	return partner, nil
}

func seedOutboundProfile(
	ctx context.Context,
	db *keel.DB,
	tenant pagination.TenantInfo,
	partner *edi.EDIPartner,
	set edi.TransactionSet,
) (*edi.EDIPartnerDocumentProfile, error) {
	template, version, err := seedBaseTemplate(ctx, db, tenant, set)
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
	if err = edidb.EDIPartnerDocumentProfiles.Create(ctx, db, profile); err != nil {
		return nil, fmt.Errorf("insert the %s document profile: %w", set, err)
	}

	return profile, nil
}

func seedBaseTemplate(
	ctx context.Context,
	db *keel.DB,
	tenant pagination.TenantInfo,
	set edi.TransactionSet,
) (*edi.EDITemplate, *edi.EDITemplateVersion, error) {
	documentType, err := edidb.EDIDocumentTypes.Query().Where(
		edidb.EDIDocumentTypes.Standard.Eq(edi.EDIStandardX12),
		edidb.EDIDocumentTypes.TransactionSet.Eq(set),
		edidb.EDIDocumentTypes.Direction.Eq(edi.DocumentDirectionOutbound),
	).First(ctx, db)
	if err != nil {
		return nil, nil, fmt.Errorf("find the X12 %s outbound document type: %w", set, err)
	}

	name := fmt.Sprintf("Base X12 %s %s", set, edi.DocumentDirectionOutbound)
	existing, existingVersion, err := findBaseTemplate(ctx, db, tenant, set, name)
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
	if err = edidb.EDITemplates.Create(ctx, db, template); err != nil {
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
	if err = edidb.EDITemplateVersions.Create(ctx, db, version); err != nil {
		return nil, nil, fmt.Errorf("insert the base %s template version: %w", set, err)
	}

	segments, err := templates.StarterSegments(tenant, version.ID, set)
	if err != nil {
		return nil, nil, err
	}
	if err = edidb.EDITemplateSegments.CreateMany(ctx, db, segments); err != nil {
		return nil, nil, fmt.Errorf("insert the base %s template segments: %w", set, err)
	}

	return template, version, nil
}

func findBaseTemplate(
	ctx context.Context,
	db *keel.DB,
	tenant pagination.TenantInfo,
	set edi.TransactionSet,
	name string,
) (*edi.EDITemplate, *edi.EDITemplateVersion, error) {
	t := edidb.EDITemplates
	template, err := t.AllTenants().Where(
		t.OrganizationID.Eq(tenant.OrgID),
		t.BusinessUnitID.Eq(tenant.BuID),
		t.Standard.Eq(edi.EDIStandardX12),
		t.TransactionSet.Eq(set),
		t.Direction.Eq(edi.DocumentDirectionOutbound),
		t.Name.Eq(name),
	).First(ctx, db)
	if errors.Is(err, keel.ErrNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("look for the base %s template: %w", set, err)
	}

	v := edidb.EDITemplateVersions
	version, err := v.AllTenants().Where(
		v.OrganizationID.Eq(tenant.OrgID),
		v.BusinessUnitID.Eq(tenant.BuID),
		v.TemplateID.Eq(template.ID),
		v.IsActive.IsTrue(),
	).First(ctx, db)
	if err != nil {
		return nil, nil, fmt.Errorf("find the active version of the base %s template: %w", set, err)
	}

	return template, version, nil
}

func seedOutboundMessages(
	ctx context.Context,
	db *keel.DB,
	tenant pagination.TenantInfo,
	partner *edi.EDIPartner,
	profile *edi.EDIPartnerDocumentProfile,
) error {
	shipments, err := shipmentdb.Shipments.AllTenants().Where(
		shipmentdb.Shipments.OrganizationID.Eq(tenant.OrgID),
		shipmentdb.Shipments.BusinessUnitID.Eq(tenant.BuID),
		shipmentdb.Shipments.CustomerID.Eq(partner.CustomerID),
	).Order(shipmentdb.Shipments.ProNumber.Asc()).Limit(seedEDIShipmentRows).List(ctx, db)
	if err != nil {
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
		if err := edidb.EDIMessages.Create(ctx, db, message); err != nil {
			return fmt.Errorf("insert 214 %d: %w", control, err)
		}
		if err := backdate(
			ctx,
			db,
			edidb.EDIMessages.Handle,
			[]string{message.ID.String()},
			[]int64{generated.Unix()},
		); err != nil {
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
