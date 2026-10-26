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
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/lading"
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
	Description: "Seeds an outside EDI trading partner for Acme Manufacturing with an active Lading project from the carrier starter and 214s in every acknowledgment state",
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
	version, err := seedLadingProject(ctx, db, tenant, partner, lading.Org{
		Name: org.Name,
		SCAC: org.ScacCode,
		Zone: org.Timezone,
	})
	if err != nil {
		return err
	}

	return seedOutboundMessages(ctx, db, tenant, partner, version)
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

// seedLadingProject gives the partner Trenova's carrier starter, checked and tested, as the
// version it runs on.
func seedLadingProject(
	ctx context.Context,
	db *keel.DB,
	tenant pagination.TenantInfo,
	partner *edi.EDIPartner,
	org lading.Org,
) (*edi.LadingVersion, error) {
	client := lading.Default()
	starter, err := client.Starter(ctx, &lading.StarterRequest{
		Name:  SeedEDIPartnerCode,
		Label: SeedEDIPartnerName,
		Role:  lading.RoleCarrier,
		Them:  &[2]string{"ZZ", seedEDIReceiverID},
		Us:    &[2]string{"ZZ", seedEDISenderID},
		Zone:  partner.Timezone,
		Usage: "production",
	})
	if err != nil {
		return nil, fmt.Errorf("write the carrier starter: %w", err)
	}
	encoded, err := lading.Encode(
		&lading.Project{Files: starter.Files, Samples: starter.Samples},
		&lading.Context{Org: org},
	)
	if err != nil {
		return nil, err
	}
	check, err := client.Check(ctx, encoded)
	if err != nil {
		return nil, fmt.Errorf("check the carrier starter: %w", err)
	}
	report, err := client.Test(ctx, encoded)
	if err != nil {
		return nil, fmt.Errorf("test the carrier starter: %w", err)
	}
	if !report.Readiness.Ready {
		return nil, fmt.Errorf("the carrier starter is not ready: %+v", report.Readiness)
	}

	project := &edi.LadingProject{
		BusinessUnitID:    tenant.BuID,
		OrganizationID:    tenant.OrgID,
		EDIPartnerID:      partner.ID,
		Settings:          map[string]any{},
		NextVersionNumber: 2,
	}
	if err = edidb.LadingProjects.Create(ctx, db, project); err != nil {
		return nil, fmt.Errorf("insert the Lading project: %w", err)
	}

	refs := make([]edi.LadingSampleRef, 0, len(starter.Samples))
	for _, sample := range starter.Samples {
		digest := lading.Digest(sample.Content)
		refs = append(refs, edi.LadingSampleRef{
			Path:   sample.Path,
			Digest: digest,
			Size:   len(sample.Content),
		})
		if err = edidb.LadingSamples.Create(ctx, db, &edi.LadingSample{
			BusinessUnitID: tenant.BuID,
			OrganizationID: tenant.OrgID,
			ProjectID:      project.ID,
			Digest:         digest,
			Size:           len(sample.Content),
			Content:        sample.Content,
		}); err != nil {
			return nil, fmt.Errorf("insert Lading sample %s: %w", sample.Path, err)
		}
	}

	now := timeutils.NowUnix()
	version := &edi.LadingVersion{
		BusinessUnitID: tenant.BuID,
		OrganizationID: tenant.OrgID,
		ProjectID:      project.ID,
		Number:         1,
		Status:         edi.LadingVersionStatusActive,
		Files:          starter.Files,
		Samples:        refs,
		Manifest:       *check,
		Readiness: &edi.LadingReadiness{
			Errors:               report.Readiness.Errors,
			Failing:              report.Readiness.Failing,
			Tests:                report.Readiness.Tests,
			UncoveredFlowClauses: report.Readiness.UncoveredFlowClauses,
			Ready:                true,
			CheckedAt:            now,
		},
		Notes:        "Trenova's carrier starter",
		ErrorCount:   check.Errors,
		WarningCount: check.Warnings,
		CertifiedAt:  &now,
		ActivatedAt:  &now,
	}
	if err = edidb.LadingVersions.Create(ctx, db, version); err != nil {
		return nil, fmt.Errorf("insert the Lading version: %w", err)
	}
	project.ActiveVersionID = version.ID
	if err = edidb.LadingProjects.Save(ctx, db, project); err != nil {
		return nil, fmt.Errorf("activate the Lading version: %w", err)
	}
	return version, nil
}

func seedOutboundMessages(
	ctx context.Context,
	db *keel.DB,
	tenant pagination.TenantInfo,
	partner *edi.EDIPartner,
	version *edi.LadingVersion,
) error {
	documentType, err := edidb.EDIDocumentTypes.Where(
		edidb.EDIDocumentTypes.Standard.Eq(edi.EDIStandardX12),
		edidb.EDIDocumentTypes.TransactionSet.Eq(edi.TransactionSet214),
		edidb.EDIDocumentTypes.Direction.Eq(edi.DocumentDirectionOutbound),
	).First(ctx, db)
	if err != nil {
		return fmt.Errorf("find the outbound 214 document type: %w", err)
	}

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
			DocumentTypeID:           documentType.ID,
			LadingVersionID:          version.ID,
			ShipmentID:               load.ID,
			Direction:                edi.DocumentDirectionOutbound,
			Standard:                 edi.EDIStandardX12,
			TransactionSet:           entry.set,
			X12Version:               edi.DefaultX12204Version,
			Status:                   edi.MessageStatusGenerated,
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
