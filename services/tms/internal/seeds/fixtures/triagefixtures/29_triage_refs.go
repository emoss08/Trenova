package triagefixtures

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	BrokerCode       = "BLRG"
	DriverEmail      = "carlos.rivera@example.com"
	OpenLanePro      = "SEED-SHP-001"
	seedDetentionPro = "SEED-DET-008"
)

const (
	inboundSeedMinute = int64(60)
	inboundSeedHour   = 60 * inboundSeedMinute
	inboundSeedDay    = 24 * inboundSeedHour

	SeedInboundMailboxToken         = "trenova-development-inbound-token"
	SeedInboundMailboxAddress       = "intake@dev.trenova.app"
	SeedInboundMailboxSigningSecret = "whsec_dHJlbm92YS1kZXZlbG9wbWVudC1pbmJvdW5kLXNpZ25pbmcta2V5" // #nosec G101 -- a published development fixture, not a credential.

	seedBillingMailboxToken   = "trenova-development-billing-token"
	seedBillingMailboxAddress = "ap@dev.trenova.app"
	seedClaimsMailboxToken    = "trenova-development-claims-token"
	seedClaimsMailboxAddress  = "claims@dev.trenova.app"
)

var seedRoutinePros = []string{
	"SEED-PAY-001",
	"SEED-PAY-002",
	"SEED-SHP-002",
	"SEED-SHP-003",
	"SEED-SHP-005",
}

var ShipmentPros = append(
	append([]string(nil), seedRoutinePros...),
	OpenLanePro,
	seedDetentionPro,
)

type Refs struct {
	Org       *tenant.Organization
	Admin     *tenant.User
	Team      Team
	Customers map[string]*customer.Customer
	Broker    *customer.Customer
	Carriers  map[string]*carrier.Carrier
	Driver    *worker.Worker
	Lane      *Lane
	Shipments map[string]*shipment.Shipment
	DocTypes  map[string]*documenttype.DocumentType
	Loc       *time.Location
	Now       int64
	DayStart  int64
	WeekStart int64
}

type Team struct {
	Dana   *tenant.User
	Marcus *tenant.User
	Priya  *tenant.User
}

type Mailboxes struct {
	Intake  *inboundmessage.Mailbox
	Billing *inboundmessage.Mailbox
	Claims  *inboundmessage.Mailbox
}

func MailboxesFor(refs *Refs) *Mailboxes {
	mailbox := func(name, address, token, purpose string) *inboundmessage.Mailbox {
		return &inboundmessage.Mailbox{
			ID:             pulid.MustNew("imbx_"),
			OrganizationID: refs.Org.ID,
			BusinessUnitID: refs.Org.BusinessUnitID,
			Name:           name,
			Address:        address,
			Provider:       inboundmessage.ProviderResend,
			TokenHash:      hashutils.SHA256Hex(token),
			Purpose:        purpose,
			ReviewPolicy:   inboundmessage.ReviewBelowConfidence,
			MinConfidence:  0.75,
			Status:         inboundmessage.MailboxActive,
		}
	}

	intake := mailbox("Intake", SeedInboundMailboxAddress, SeedInboundMailboxToken,
		"Tenders, rate confirmations, PODs and status requests")
	intake.BlockedSenders = []string{"deals.loadblaster.example"}

	billing := mailbox("Billing", seedBillingMailboxAddress, seedBillingMailboxToken,
		"Carrier invoices and payment questions")
	billing.MinConfidence = 0.85
	billing.TestSentAt = refs.Now - 3*inboundSeedDay
	billing.TestReceivedAt = refs.Now - 3*inboundSeedDay + 40

	claims := mailbox("Claims", seedClaimsMailboxAddress, seedClaimsMailboxToken,
		"Detention disputes, shortages and damage claims")
	claims.ReviewPolicy = inboundmessage.ReviewAlways
	claims.MinConfidence = 0
	claims.LastReceivedAt = refs.Now - 7*inboundSeedDay - 2*inboundSeedHour

	return &Mailboxes{Intake: intake, Billing: billing, Claims: claims}
}

var CarrierCodes = []string{"IHF", "HPT", "LSH"}

type Lane struct {
	Shipment         *shipment.Shipment
	Origin           string
	OriginCity       string
	OriginState      string
	Destination      string
	DestinationCity  string
	DestinationState string
	PickupAt         int64
}

func (l *Lane) pickupDay(loc *time.Location) string {
	return time.Unix(l.PickupAt, 0).In(loc).Format(time.DateOnly)
}

func (r *Refs) correctedAt(index int) int64 {
	span := r.Now - r.WeekStart
	return r.WeekStart + span*int64(index+1)/int64(len(seedCorrectionsThisWeek)+1)
}

func (r *Refs) lastWeekCorrectedAt() int64 {
	return r.WeekStart - 3*inboundSeedDay + 10*inboundSeedHour
}
