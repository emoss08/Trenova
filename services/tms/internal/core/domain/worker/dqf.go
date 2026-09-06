package worker

import (
	"sort"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/documentpacketrule"
	"github.com/emoss08/trenova/shared/pulid"
)

// DefaultDQFRetentionDays is how long a driver qualification file is held past
// termination before it may be purged: the three years 49 CFR 391.51(d)
// requires. It is a default, not a limit — an organisation can hold longer.
const DefaultDQFRetentionDays = 1095

// DQFSection groups a requirement by where its evidence lives, so the file
// reads as a set of areas rather than a flat list of forty rows.
type DQFSection string

const (
	DQFSectionCredentials   = DQFSection("Credentials")
	DQFSectionDocuments     = DQFSection("Documents")
	DQFSectionSafetyHistory = DQFSection("SafetyHistory")
	DQFSectionDrugAlcohol   = DQFSection("DrugAlcohol")
)

func (s DQFSection) String() string { return string(s) }

// DQFItemStatus is one requirement's state. It deliberately mirrors the words
// the credential and document areas already use, so a reader who knows one
// knows this.
type DQFItemStatus string

const (
	DQFSatisfied     = DQFItemStatus("Satisfied")
	DQFExpiringSoon  = DQFItemStatus("ExpiringSoon")
	DQFExpired       = DQFItemStatus("Expired")
	DQFMissing       = DQFItemStatus("Missing")
	DQFOutstanding   = DQFItemStatus("Outstanding")
	DQFNotApplicable = DQFItemStatus("NotApplicable")
)

func (s DQFItemStatus) String() string { return string(s) }

// Blocks reports whether this item stops the file being complete. Expiring soon
// warns: the document on file is still valid today, and treating it as a gap
// would make every file incomplete for a month before every renewal.
func (s DQFItemStatus) Blocks() bool {
	return s == DQFExpired || s == DQFMissing || s == DQFOutstanding
}

// DQFItem is one line of the file.
type DQFItem struct {
	Section DQFSection
	// Code is stable and safe to key on: a credential type code, a document
	// type code, or one of the fixed codes below.
	Code       string
	Name       string
	Status     DQFItemStatus
	Detail     string
	Regulation string
	ExpiresAt  *int64
}

// The fixed item codes, for the requirements that are not a credential type or
// a document type.
const (
	DQFCodeSafetyHistory      = "safety_performance_history"
	DQFCodePreEmploymentTest  = "pre_employment_drug_test"
	DQFCodePreEmploymentQuery = "pre_employment_clearinghouse_query"
	DQFCodeDrugAlcoholRecord  = "drug_alcohol_standing"
)

// DQFFile is a driver's qualification file as 49 CFR 391.51 asks for it,
// assembled from the areas that already own each piece rather than copied into
// a file of its own. Nothing here is stored: it is derived on read, so it can
// never drift from the credentials, documents and investigations underneath.
type DQFFile struct {
	WorkerID        pulid.ID
	Complete        bool
	Items           []DQFItem
	MissingRequired int
	ExpiringSoon    int
	Expired         int
	Outstanding     int

	HireDate        int64
	TerminationDate *int64

	// SafetyHistoryDueAt is thirty days after hire, when the previous-employer
	// investigation had to be complete (49 CFR 391.23(c)(1)). Zero when the
	// hire date is unknown.
	SafetyHistoryDueAt int64
	SafetyHistoryLate  bool

	// RetentionExpiresAt is when the file may be purged; nil while the driver
	// is employed, because the clock runs from termination.
	RetentionExpiresAt *int64
	PurgeEligible      bool
}

// DQFInput is the evidence the file is assembled from. Every field may be
// absent: a tenant with no document packet rules configured simply has no
// document section, which is a fair report of the truth rather than a pile of
// invented gaps.
type DQFInput struct {
	WorkerID        pulid.ID
	HireDate        int64
	TerminationDate *int64
	Credentials     *WorkerCredentialSummary
	Documents       *documentpacketrule.PacketSummary
	Verifications   []*WorkerEmploymentVerification
	DrugAlcohol     *DrugAlcoholStanding
	// RetentionDays is the organisation's setting; zero falls back to the
	// regulation's three years.
	RetentionDays int
	Now           int64
}

// BuildDQF assembles the file. It reports what the record says and never
// guesses: a section with no evidence configured contributes no items, and an
// item is only Missing when something that should exist does not.
func BuildDQF(in DQFInput) *DQFFile {
	file := &DQFFile{
		WorkerID:        in.WorkerID,
		HireDate:        in.HireDate,
		TerminationDate: in.TerminationDate,
		Items:           make([]DQFItem, 0, 16),
	}

	file.addCredentials(in.Credentials)
	file.addDocuments(in.Documents)
	file.addSafetyHistory(in.Verifications, in.HireDate, in.Now)
	file.addDrugAlcohol(in.DrugAlcohol)

	sort.SliceStable(file.Items, func(i, j int) bool {
		a, b := file.Items[i], file.Items[j]
		if a.Section != b.Section {
			return sectionRank(a.Section) < sectionRank(b.Section)
		}
		return statusRank(a.Status) < statusRank(b.Status)
	})

	for _, item := range file.Items {
		switch item.Status {
		case DQFMissing:
			file.MissingRequired++
		case DQFExpired:
			file.Expired++
		case DQFExpiringSoon:
			file.ExpiringSoon++
		case DQFOutstanding:
			file.Outstanding++
		case DQFSatisfied, DQFNotApplicable:
		}
	}
	file.Complete = file.MissingRequired == 0 && file.Expired == 0 && file.Outstanding == 0

	file.applyRetention(in.RetentionDays, in.Now)

	return file
}

func (f *DQFFile) addCredentials(summary *WorkerCredentialSummary) {
	if summary == nil {
		return
	}
	for _, item := range summary.Items {
		// Optional credentials a worker happens to hold are not part of the
		// qualification file; listing them would pad an audit with cards
		// nobody is required to have.
		if item == nil || !item.Required || item.CredentialType == nil {
			continue
		}
		entry := DQFItem{
			Section:    DQFSectionCredentials,
			Code:       item.CredentialType.Code,
			Name:       item.CredentialType.Name,
			Status:     dqfStatusForHealth(item.Health),
			Regulation: item.CredentialType.Description,
		}
		if item.Credential != nil {
			entry.ExpiresAt = item.Credential.ExpiresAt
		}
		entry.Detail = credentialDetail(item)
		f.Items = append(f.Items, entry)
	}
}

func credentialDetail(item *CredentialSummaryItem) string {
	switch item.Health {
	case CredentialHealthMissing:
		return "Nothing on file."
	case CredentialHealthExpired:
		return "Expired; the file is not complete until it is renewed."
	case CredentialHealthExpiringSoon:
		if item.DaysUntilExpiry != nil {
			return "Expires in " + strconv.FormatInt(*item.DaysUntilExpiry, 10) + " days."
		}
		return "Expiring soon."
	default:
		return "On file."
	}
}

func (f *DQFFile) addDocuments(summary *documentpacketrule.PacketSummary) {
	if summary == nil {
		return
	}
	for _, item := range summary.Items {
		if !item.Required {
			continue
		}
		f.Items = append(f.Items, DQFItem{
			Section: DQFSectionDocuments,
			Code:    item.DocumentTypeCode,
			Name:    item.DocumentTypeName,
			Status:  dqfStatusForPacketItem(item.Status),
			Detail:  documentDetail(item),
		})
	}
}

func documentDetail(item documentpacketrule.PacketItemSummary) string {
	switch item.Status {
	case documentpacketrule.ItemStatusMissing:
		return "No document uploaded."
	case documentpacketrule.ItemStatusExpired:
		return "The document on file has expired."
	case documentpacketrule.ItemStatusExpiringSoon:
		return "The document on file expires soon."
	case documentpacketrule.ItemStatusNeedsReview:
		return "Uploaded and waiting on review."
	default:
		return "On file."
	}
}

// addSafetyHistory reports the 391.23 investigation as one item. It is a single
// line because that is how an auditor asks the question — "did you investigate
// the previous employers?" — and the per-employer detail lives beside it.
func (f *DQFFile) addSafetyHistory(
	verifications []*WorkerEmploymentVerification,
	hireDate int64,
	now int64,
) {
	item := DQFItem{
		Section:    DQFSectionSafetyHistory,
		Code:       DQFCodeSafetyHistory,
		Name:       "Previous employer safety performance history",
		Regulation: "49 CFR 391.23, 382.413",
	}

	if hireDate > 0 {
		f.SafetyHistoryDueAt = DueAtForHire(hireDate)
	}

	if len(verifications) == 0 {
		// No employers recorded is not the same as none existing. Until
		// somebody says the driver had no DOT-regulated employment, the
		// investigation has not been made.
		item.Status = DQFMissing
		item.Detail = "No previous employer has been recorded for the three-year lookback."
		f.markSafetyHistoryLate(now)
		f.Items = append(f.Items, item)
		return
	}

	outstanding, awaitingDrugAlcohol := 0, 0
	for _, verification := range verifications {
		if verification == nil {
			continue
		}
		if verification.IsOutstanding() {
			outstanding++
		}
		if verification.Status == VerificationReceived && verification.NeedsDrugAlcoholAnswer() {
			awaitingDrugAlcohol++
		}
	}

	switch {
	case outstanding > 0:
		item.Status = DQFOutstanding
		item.Detail = plural(outstanding, "employer", "employers") + " still awaiting a response."
		f.markSafetyHistoryLate(now)
	case awaitingDrugAlcohol > 0:
		item.Status = DQFOutstanding
		item.Detail = plural(awaitingDrugAlcohol, "employer", "employers") +
			" answered without the drug and alcohol history (49 CFR 382.413)."
		f.markSafetyHistoryLate(now)
	default:
		item.Status = DQFSatisfied
		item.Detail = plural(len(verifications), "previous employer", "previous employers") + " investigated."
	}

	f.Items = append(f.Items, item)
}

func (f *DQFFile) markSafetyHistoryLate(now int64) {
	if f.SafetyHistoryDueAt > 0 && now > f.SafetyHistoryDueAt {
		f.SafetyHistoryLate = true
	}
}

// addDrugAlcohol reports the two gates a driver must clear before their first
// dispatch. The standing itself is reported too, because a prohibition belongs
// in the file even though it is not a missing document.
func (f *DQFFile) addDrugAlcohol(standing *DrugAlcoholStanding) {
	if standing == nil {
		return
	}

	testStatus := DQFMissing
	testDetail := "No passed pre-employment drug test is on file."
	if standing.HasPreEmploymentTest {
		testStatus = DQFSatisfied
		testDetail = "Passed and on file."
	}
	f.Items = append(f.Items, DQFItem{
		Section:    DQFSectionDrugAlcohol,
		Code:       DQFCodePreEmploymentTest,
		Name:       "Pre-employment drug test",
		Status:     testStatus,
		Detail:     testDetail,
		Regulation: "49 CFR 382.301(a)",
	})

	queryStatus := DQFMissing
	queryDetail := "No pre-employment full query is on file."
	if standing.HasPreEmploymentQuery {
		queryStatus = DQFSatisfied
		queryDetail = "Run and returned clear."
	}
	f.Items = append(f.Items, DQFItem{
		Section:    DQFSectionDrugAlcohol,
		Code:       DQFCodePreEmploymentQuery,
		Name:       "Pre-employment Clearinghouse query",
		Status:     queryStatus,
		Detail:     queryDetail,
		Regulation: "49 CFR 382.701(a)",
	})

	if standing.Status == DrugAlcoholProhibited {
		f.Items = append(f.Items, DQFItem{
			Section:    DQFSectionDrugAlcohol,
			Code:       DQFCodeDrugAlcoholRecord,
			Name:       "Drug and alcohol standing",
			Status:     DQFOutstanding,
			Detail:     "Prohibited from safety-sensitive duty until the return-to-duty process is complete.",
			Regulation: "49 CFR 382.501",
		})
	}
}

// applyRetention works out when the file may be purged. The clock runs from
// termination, so an employed driver's file has no expiry — and the file is
// only ever flagged, never deleted here.
func (f *DQFFile) applyRetention(retentionDays int, now int64) {
	if f.TerminationDate == nil || *f.TerminationDate <= 0 {
		return
	}
	days := retentionDays
	if days <= 0 {
		days = DefaultDQFRetentionDays
	}
	expires := *f.TerminationDate + int64(days)*secondsPerDay
	f.RetentionExpiresAt = &expires
	f.PurgeEligible = now >= expires
}

func dqfStatusForHealth(health CredentialHealth) DQFItemStatus {
	switch health {
	case CredentialHealthValid:
		return DQFSatisfied
	case CredentialHealthExpiringSoon:
		return DQFExpiringSoon
	case CredentialHealthExpired:
		return DQFExpired
	case CredentialHealthMissing:
		return DQFMissing
	default:
		return DQFMissing
	}
}

func dqfStatusForPacketItem(status documentpacketrule.ItemStatus) DQFItemStatus {
	switch status {
	case documentpacketrule.ItemStatusComplete:
		return DQFSatisfied
	case documentpacketrule.ItemStatusExpiringSoon:
		return DQFExpiringSoon
	case documentpacketrule.ItemStatusExpired:
		return DQFExpired
	case documentpacketrule.ItemStatusNeedsReview:
		return DQFOutstanding
	case documentpacketrule.ItemStatusMissing:
		return DQFMissing
	default:
		return DQFMissing
	}
}

func sectionRank(section DQFSection) int {
	switch section {
	case DQFSectionCredentials:
		return 0
	case DQFSectionDocuments:
		return 1
	case DQFSectionSafetyHistory:
		return 2
	case DQFSectionDrugAlcohol:
		return 3
	default:
		return 4
	}
}

func statusRank(status DQFItemStatus) int {
	switch status {
	case DQFMissing:
		return 0
	case DQFExpired:
		return 1
	case DQFOutstanding:
		return 2
	case DQFExpiringSoon:
		return 3
	case DQFSatisfied:
		return 4
	default:
		return 5
	}
}
