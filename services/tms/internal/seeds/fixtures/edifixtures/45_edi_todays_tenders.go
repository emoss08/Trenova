package edifixtures

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument/edidocumentdb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile/ediinboundfiledb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage/edimessagedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer/editransferdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	SeedTenderProfileName   = "Acme Manufacturing SFTP mailbox"
	SeedTenderFilePrefix    = "ACMEMFG_TENDER_"
	SeedTenderQuarantine    = "ship-to 0412 is not a known location"
	SeedTenderLoginRefused  = "SFTP login refused"
	seedTenderInboundDir    = "/outbound/to-trnv"
	seedTenderDeliveryTries = 3
)

type seedTender struct {
	bol        string
	age        time.Duration
	quarantine bool
	answer     edimessage.MessageDeliveryStatus
	ack        edimessage.MessageAcknowledgmentStatus
	answerIn   time.Duration
}

var seedTenders = []seedTender{
	{
		bol: "88214", age: 3*time.Hour + 39*time.Minute, answerIn: 3 * time.Minute,
		answer: edimessage.MessageDeliveryStatusSent,
		ack:    edimessage.MessageAcknowledgmentStatusAccepted,
	},
	{
		bol: "88215", age: 3*time.Hour + 38*time.Minute + 40*time.Second, answerIn: 2 * time.Minute,
		answer: edimessage.MessageDeliveryStatusSent,
		ack:    edimessage.MessageAcknowledgmentStatusAccepted,
	},
	{bol: "88231", age: 2*time.Hour + time.Minute, quarantine: true},
	{
		bol: "88240", age: time.Hour + 26*time.Minute, answerIn: 4 * time.Minute,
		answer: edimessage.MessageDeliveryStatusSent,
		ack:    edimessage.MessageAcknowledgmentStatusPending,
	},
	{
		bol: "88243", age: 49 * time.Minute, answerIn: 3 * time.Minute,
		answer: edimessage.MessageDeliveryStatusSent,
		ack:    edimessage.MessageAcknowledgmentStatusAccepted,
	},
	{
		bol: "88244", age: 36 * time.Minute, answerIn: 2 * time.Minute,
		answer: edimessage.MessageDeliveryStatusFailed,
		ack:    edimessage.MessageAcknowledgmentStatusPending,
	},
}

// EDITodaysTendersSeed gives the Acme Manufacturing partner a morning of load tenders: six 204s
// over SFTP, one quarantined, five that became shipments and were answered with a 990, one of
// which never reached Acme. Bench scenarios move the morning into the day they run.
type EDITodaysTendersSeed struct{}

type tenderSeedRun struct {
	db        *keel.DB
	tenant    pagination.TenantInfo
	partner   *edi.EDIPartner
	profile   *edi.EDICommunicationProfile
	tender204 *edidocument.EDIDocumentType
	answer990 *edidocument.EDIDocumentType
	shipments []*shipment.Shipment
	now       time.Time
}

func (s *EDITodaysTendersSeed) Run(ctx context.Context, sr *seed.Run) error {
	org, err := tenantfixtures.DefaultOrganization(ctx, sr)
	if err != nil {
		return err
	}
	run := &tenderSeedRun{
		db:     sr.DB,
		tenant: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID},
		now:    time.Now().UTC(),
	}
	seeded, err := run.alreadySeeded(ctx)
	if err != nil || seeded {
		return err
	}
	if err = run.load(ctx); err != nil {
		return err
	}
	if err = run.createProfile(ctx); err != nil {
		return err
	}
	for idx := range seedTenders {
		if err = run.createTender(ctx, idx, &seedTenders[idx]); err != nil {
			return fmt.Errorf("seed tender %s: %w", seedTenders[idx].bol, err)
		}
	}

	return nil
}

func (r *tenderSeedRun) alreadySeeded(ctx context.Context) (bool, error) {
	f := ediinboundfiledb.EDIInboundFiles

	return f.AllTenants().Where(
		f.OrganizationID.Eq(r.tenant.OrgID),
		f.BusinessUnitID.Eq(r.tenant.BuID),
		f.FileName.HasPrefix(SeedTenderFilePrefix),
	).Exists(ctx, r.db)
}

func (r *tenderSeedRun) load(ctx context.Context) error {
	p := edidb.EDIPartners
	partner, err := p.AllTenants().Where(
		p.OrganizationID.Eq(r.tenant.OrgID),
		p.BusinessUnitID.Eq(r.tenant.BuID),
		p.Code.Eq(SeedEDIPartnerCode),
	).First(ctx, r.db)
	if err != nil {
		return fmt.Errorf("find the %s partner: %w", SeedEDIPartnerCode, err)
	}
	r.partner = partner

	if r.tender204, err = documentType(ctx, r.db, edidocument.TransactionSet204,
		edidocument.DocumentDirectionInbound); err != nil {
		return err
	}
	if r.answer990, err = documentType(ctx, r.db, edidocument.TransactionSet990,
		edidocument.DocumentDirectionOutbound); err != nil {
		return err
	}

	s := shipmentdb.Shipments
	r.shipments, err = s.AllTenants().Where(
		s.OrganizationID.Eq(r.tenant.OrgID),
		s.BusinessUnitID.Eq(r.tenant.BuID),
		s.CustomerID.Eq(partner.CustomerID),
	).Order(s.ProNumber.Desc()).Limit(len(seedTenders)).List(ctx, r.db)
	if err != nil {
		return fmt.Errorf("read Acme Manufacturing's shipments: %w", err)
	}
	if len(r.shipments) == 0 {
		return errors.New("the development seed has no Acme Manufacturing shipment for a tender")
	}

	return nil
}

func documentType(
	ctx context.Context,
	db *keel.DB,
	set edidocument.TransactionSet,
	direction edidocument.DocumentDirection,
) (*edidocument.EDIDocumentType, error) {
	d := edidocumentdb.EDIDocumentTypes
	found, err := d.Where(
		d.Standard.Eq(edidocument.EDIStandardX12),
		d.TransactionSet.Eq(set),
		d.Direction.Eq(direction),
	).First(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("find the %s %s document type: %w", direction, set, err)
	}

	return found, nil
}

func (r *tenderSeedRun) createProfile(ctx context.Context) error {
	c := edidb.EDICommunicationProfiles
	existing, err := c.AllTenants().Where(
		c.OrganizationID.Eq(r.tenant.OrgID),
		c.BusinessUnitID.Eq(r.tenant.BuID),
		c.Name.Eq(SeedTenderProfileName),
	).First(ctx, r.db)
	if err == nil {
		r.profile = existing
		return nil
	}
	if !errors.Is(err, keel.ErrNotFound) {
		return err
	}
	r.profile = &edi.EDICommunicationProfile{
		BusinessUnitID: r.tenant.BuID,
		OrganizationID: r.tenant.OrgID,
		EDIPartnerID:   r.partner.ID,
		Method:         edi.ConnectionMethodSFTP,
		Status:         domaintypes.StatusActive,
		Name:           SeedTenderProfileName,
		Description:    "Acme Manufacturing's SFTP server: their 204s in, our 990s out.",
		Config: map[string]any{
			"host":              "sftp.acme-mfg.example",
			"port":              "22",
			"username":          "trnv",
			"authMode":          "password",
			"environment":       "test",
			"inboundDirectory":  seedTenderInboundDir,
			"outboundDirectory": "/inbound/from-trnv",
		},
		EncryptedSecrets: map[string]string{},
	}

	return edidb.EDICommunicationProfiles.Create(ctx, r.db, r.profile)
}

func (r *tenderSeedRun) createTender(ctx context.Context, idx int, entry *seedTender) error {
	received := r.now.Add(-entry.age)
	raw := seed204(received, entry.bol)
	fileName := SeedTenderFilePrefix + entry.bol + ".edi"
	processed := received.Add(20 * time.Second).Unix()
	digest := sha256.Sum256([]byte(raw))
	file := &ediinboundfile.EDIInboundFile{
		BusinessUnitID:           r.tenant.BuID,
		OrganizationID:           r.tenant.OrgID,
		CommunicationProfileID:   r.profile.ID,
		EDIPartnerID:             r.partner.ID,
		Method:                   edi.ConnectionMethodSFTP,
		RemotePath:               seedTenderInboundDir + "/" + fileName,
		FileName:                 fileName,
		Checksum:                 hex.EncodeToString(digest[:]),
		SizeBytes:                int64(len(raw)),
		RawContent:               raw,
		InterchangeControlNumber: interchange(entry.bol),
		ISASenderQualifier:       "ZZ",
		ISASenderID:              seedEDIReceiverID,
		ISAReceiverQualifier:     "ZZ",
		ISAReceiverID:            seedEDISenderID,
		Status:                   ediinboundfile.InboundFileStatusProcessed,
		TransactionCount:         1,
		ReceivedAt:               received.Unix(),
		ProcessedAt:              &processed,
	}
	if entry.quarantine {
		file.Status = ediinboundfile.InboundFileStatusQuarantined
		file.FailureReason = SeedTenderQuarantine
	}
	if err := ediinboundfiledb.EDIInboundFiles.Create(ctx, r.db, file); err != nil {
		return fmt.Errorf("insert the inbound file: %w", err)
	}

	tender := r.tenderMessage(file, entry, raw)
	if err := edimessagedb.EDIMessages.Create(ctx, r.db, tender); err != nil {
		return fmt.Errorf("insert the 204: %w", err)
	}
	if entry.quarantine {
		return r.backdate(ctx, received, file.ID, tender.ID)
	}

	load := r.shipments[idx%len(r.shipments)]
	transfer := r.transfer(tender, load, entry, received)
	if err := editransferdb.EDITransfers.Create(ctx, r.db, transfer); err != nil {
		return fmt.Errorf("insert the tender: %w", err)
	}
	answer := r.answerMessage(transfer, load, entry, received)
	if err := edimessagedb.EDIMessages.Create(ctx, r.db, answer); err != nil {
		return fmt.Errorf("insert the 990: %w", err)
	}

	return r.backdate(ctx, received, file.ID, tender.ID)
}

func (r *tenderSeedRun) backdate(
	ctx context.Context,
	received time.Time,
	fileID, tenderID pulid.ID,
) error {
	if err := tenantfixtures.Backdate(ctx, r.db, ediinboundfiledb.EDIInboundFiles,
		[]string{fileID.String()}, []int64{received.Unix()}); err != nil {
		return fmt.Errorf("date the inbound file: %w", err)
	}

	return tenantfixtures.Backdate(ctx, r.db, edimessagedb.EDIMessages,
		[]string{tenderID.String()}, []int64{received.Unix()})
}

func (r *tenderSeedRun) tenderMessage(
	file *ediinboundfile.EDIInboundFile,
	entry *seedTender,
	raw string,
) *edimessage.EDIMessage {
	status := edimessage.MessageStatusGenerated
	if entry.quarantine {
		status = edimessage.MessageStatusFailed
	}

	return &edimessage.EDIMessage{
		BusinessUnitID:           r.tenant.BuID,
		OrganizationID:           r.tenant.OrgID,
		EDIPartnerID:             r.partner.ID,
		DocumentTypeID:           r.tender204.ID,
		InboundFileID:            file.ID,
		Direction:                edidocument.DocumentDirectionInbound,
		Standard:                 edidocument.EDIStandardX12,
		TransactionSet:           edidocument.TransactionSet204,
		X12Version:               edidocument.DefaultX12204Version,
		Status:                   status,
		InterchangeControlNumber: file.InterchangeControlNumber,
		GroupControlNumber:       entry.bol,
		TransactionControlNumber: "0001",
		SegmentCount:             int64(strings.Count(raw, "~") - 4),
		RawX12:                   raw,
		PayloadSnapshot: edipayload.DocumentPayload{
			TransactionSet: edidocument.TransactionSet204,
		},
		AckStatus:   edimessage.MessageAcknowledgmentStatusNotExpected,
		GeneratedAt: file.ReceivedAt,
	}
}

func (r *tenderSeedRun) transfer(
	tender *edimessage.EDIMessage,
	load *shipment.Shipment,
	entry *seedTender,
	received time.Time,
) *editransfer.EDITransfer {
	approved := received.Add(time.Minute).Unix()

	return &editransfer.EDITransfer{
		SourceOrganizationID: r.tenant.OrgID,
		SourceBusinessUnitID: r.tenant.BuID,
		TargetOrganizationID: r.tenant.OrgID,
		TargetBusinessUnitID: r.tenant.BuID,
		SourcePartnerID:      r.partner.ID,
		TargetPartnerID:      r.partner.ID,
		TargetShipmentID:     load.ID,
		InboundMessageID:     tender.ID,
		Status:               editransfer.TransferStatusApproved,
		TenderPayload: editransfer.LoadTenderPayload{
			PurposeCode:    editransfer.LoadTenderPurposeOriginal,
			BusinessUnitID: r.tenant.BuID,
			OrganizationID: r.tenant.OrgID,
			CustomerID:     r.partner.CustomerID,
			BOL:            entry.bol,
			RatingDetail:   map[string]any{"externalShipmentId": entry.bol},
			Moves:          []editransfer.LoadTenderMove{},
			Commodities:    []editransfer.LoadTenderCommodity{},
		},
		MappingSnapshot: []editransfer.MappingResolution{},
		SubmittedAt:     received.Add(20 * time.Second).Unix(),
		ApprovedAt:      &approved,
		ProcessedAt:     &approved,
	}
}

func (r *tenderSeedRun) answerMessage(
	transfer *editransfer.EDITransfer,
	load *shipment.Shipment,
	entry *seedTender,
	received time.Time,
) *edimessage.EDIMessage {
	generated := received.Add(entry.answerIn - 30*time.Second)
	attempted := received.Add(entry.answerIn).Unix()
	message := &edimessage.EDIMessage{
		BusinessUnitID:           r.tenant.BuID,
		OrganizationID:           r.tenant.OrgID,
		EDIPartnerID:             r.partner.ID,
		DocumentTypeID:           r.answer990.ID,
		ShipmentID:               load.ID,
		TransferID:               transfer.ID,
		Direction:                edidocument.DocumentDirectionOutbound,
		Standard:                 edidocument.EDIStandardX12,
		TransactionSet:           edidocument.TransactionSet990,
		X12Version:               edidocument.DefaultX12204Version,
		Status:                   edimessage.MessageStatusGenerated,
		InterchangeControlNumber: interchange("9" + entry.bol),
		GroupControlNumber:       "9" + entry.bol,
		TransactionControlNumber: "0001",
		SegmentCount:             4,
		RawX12:                   seed990(generated, entry.bol),
		PayloadSnapshot: edipayload.DocumentPayload{
			TransactionSet: edidocument.TransactionSet990,
		},
		DeliveryStatus:        entry.answer,
		DeliveryRemotePath:    "/inbound/from-trnv/TRNV_990_" + entry.bol + ".edi",
		DeliveryAttempts:      1,
		DeliveryLastAttemptAt: &attempted,
		AckStatus:             entry.ack,
		GeneratedAt:           generated.Unix(),
	}
	if entry.answer == edimessage.MessageDeliveryStatusSent {
		message.DeliverySentAt = &attempted
	} else {
		retried := r.now.Add(-6 * time.Minute).Unix()
		message.DeliveryAttempts = seedTenderDeliveryTries
		message.DeliveryLastAttemptAt = &retried
		message.DeliveryLastError = SeedTenderLoginRefused
	}
	if entry.ack == edimessage.MessageAcknowledgmentStatusAccepted {
		acked := received.Add(entry.answerIn + 6*time.Minute).Unix()
		message.AckReceivedAt = &acked
	}

	return message
}

func interchange(control string) string {
	return fmt.Sprintf("%09s", control)
}

func seed204(at time.Time, bol string) string {
	control := interchange(bol)
	segments := []string{
		fmt.Sprintf(
			"ISA*00*          *00*          *ZZ*%-15s*ZZ*%-15s*%s*%s*U*00401*%s*0*P*>",
			seedEDIReceiverID, seedEDISenderID, at.Format("060102"), at.Format("1504"), control,
		),
		fmt.Sprintf("GS*SM*%s*%s*%s*%s*%s*X*004010",
			seedEDIReceiverID, seedEDISenderID, at.Format("20060102"), at.Format("1504"), bol),
		"ST*204*0001",
		fmt.Sprintf("B2**%s**%s**PP", seedEDICarrierSCAC, bol),
		"B2A*00",
		fmt.Sprintf("L11*%s*BM", bol),
		"S5*1*LD",
		"N1*SH*ACME MANUFACTURING",
		"N4*LOS ANGELES*CA*90021",
		"S5*2*UL",
		"N1*CN*CHICAGO DISTRIBUTION CENTER",
		"N4*CHICAGO*IL*60609",
		"SE*11*0001",
		"GE*1*" + bol,
		"IEA*1*" + control,
	}

	return strings.Join(segments, "~") + "~"
}

func seed990(at time.Time, bol string) string {
	control := interchange("9" + bol)
	segments := []string{
		fmt.Sprintf(
			"ISA*00*          *00*          *ZZ*%-15s*ZZ*%-15s*%s*%s*U*00401*%s*0*P*>",
			seedEDISenderID, seedEDIReceiverID, at.Format("060102"), at.Format("1504"), control,
		),
		fmt.Sprintf("GS*GF*%s*%s*%s*%s*9%s*X*004010",
			seedEDISenderID, seedEDIReceiverID, at.Format("20060102"), at.Format("1504"), bol),
		"ST*990*0001",
		fmt.Sprintf("B1*%s*%s*%s*A", seedEDICarrierSCAC, bol, at.Format("20060102")),
		fmt.Sprintf("N9*CN*%s", bol),
		"SE*4*0001",
		"GE*1*9" + bol,
		"IEA*1*" + control,
	}

	return strings.Join(segments, "~") + "~"
}
