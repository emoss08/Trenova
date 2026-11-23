package workerpage

import (
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/pulid"
)

const recordKind = "worker"

type Tab string

const (
	TabOverview      = Tab("Overview")
	TabProfile       = Tab("Profile")
	TabEmployment    = Tab("Employment")
	TabQualification = Tab("Qualification")
	TabDevelopment   = Tab("Development")
	TabSafety        = Tab("Safety")
	TabTime          = Tab("Time")
	TabPay           = Tab("Pay")
	TabFiles         = Tab("Files")
	TabActivity      = Tab("Activity")
)

func (t Tab) String() string { return string(t) }

type View string

const (
	ViewHistory        = View("History")
	ViewChecklists     = View("Checklists")
	ViewCredentials    = View("Credentials")
	ViewDqFile         = View("DqFile")
	ViewDrugAndAlcohol = View("DrugAndAlcohol")
	ViewTraining       = View("Training")
	ViewReviews        = View("Reviews")
	ViewHoursOfService = View("HoursOfService")
	ViewSchedule       = View("Schedule")
	ViewTimeOff        = View("TimeOff")
	ViewLeave          = View("Leave")
	ViewPayProfile     = View("PayProfile")
	ViewBenefits       = View("Benefits")
	ViewDocuments      = View("Documents")
	ViewDriverPortal   = View("DriverPortal")
	ViewAgentMemory    = View("AgentMemory")
)

func (v View) String() string { return string(v) }

type Record string

const (
	RecordCredential = Record("Credential")
	RecordTraining   = Record("Training")
	RecordReview     = Record("Review")
	RecordEvent      = Record("Event")
	RecordChecklist  = Record("Checklist")
	RecordLeave      = Record("Leave")
	RecordTest       = Record("Test")
	RecordEmployer   = Record("Employer")
	RecordInjury     = Record("Injury")
	RecordRequest    = Record("Request")
)

func (r Record) String() string { return string(r) }

type Destination struct {
	Tab      Tab
	View     *View
	Record   *Record
	RecordID *pulid.ID
}

const (
	QueryTab  = "tab"
	QueryView = "view"
)

func (t Tab) Slug() string {
	switch t {
	case TabOverview:
		return "overview"
	case TabProfile:
		return "profile"
	case TabEmployment:
		return "employment"
	case TabQualification:
		return "qualification"
	case TabDevelopment:
		return "development"
	case TabSafety:
		return "safety"
	case TabTime:
		return "time"
	case TabPay:
		return "pay"
	case TabFiles:
		return "files"
	case TabActivity:
		return "activity"
	default:
		return ""
	}
}

func (v View) Slug() string {
	switch v {
	case ViewHistory:
		return "history"
	case ViewChecklists:
		return "checklists"
	case ViewCredentials:
		return "credentials"
	case ViewDqFile:
		return "dq-file"
	case ViewDrugAndAlcohol:
		return "drug-and-alcohol"
	case ViewTraining:
		return "training"
	case ViewReviews:
		return "reviews"
	case ViewHoursOfService:
		return "hours-of-service"
	case ViewSchedule:
		return "schedule"
	case ViewTimeOff:
		return "time-off"
	case ViewLeave:
		return "leave"
	case ViewPayProfile:
		return "pay-profile"
	case ViewBenefits:
		return "benefits"
	case ViewDocuments:
		return "documents"
	case ViewDriverPortal:
		return "driver-portal"
	case ViewAgentMemory:
		return "agent-memory"
	default:
		return ""
	}
}

func (r Record) Param(tab Tab) string {
	switch r {
	case RecordCredential:
		return "credential"
	case RecordTraining:
		return "training"
	case RecordReview:
		return "review"
	case RecordEvent:
		if tab == TabSafety {
			return "safetyEvent"
		}
		return "event"
	case RecordChecklist:
		return "checklist"
	case RecordLeave:
		return "leave"
	case RecordTest:
		return "test"
	case RecordEmployer:
		return "employer"
	case RecordInjury:
		return "injury"
	case RecordRequest:
		if tab == TabTime {
			return "timeOff"
		}
		return "request"
	default:
		return ""
	}
}

func At(tab Tab) Destination {
	return Destination{Tab: tab}
}

func AtView(tab Tab, view View) Destination {
	return Destination{Tab: tab, View: &view}
}

func (d Destination) WithRecord(record Record, id pulid.ID) Destination {
	if id.IsNil() {
		return d
	}
	d.Record = &record
	d.RecordID = &id
	return d
}

func (d Destination) Query() map[string]string {
	query := make(map[string]string, 3)
	if slug := d.Tab.Slug(); slug != "" && d.Tab != TabOverview {
		query[QueryTab] = slug
	}
	if d.View != nil {
		if slug := d.View.Slug(); slug != "" {
			query[QueryView] = slug
		}
	}
	if d.Record != nil && d.RecordID != nil && d.RecordID.IsNotNil() {
		if param := d.Record.Param(d.Tab); param != "" {
			query[param] = d.RecordID.String()
		}
	}
	return query
}

func Path(workerID pulid.ID, dest Destination) string {
	path, _ := productguide.Default.RecordPath(recordKind, workerID.String(), dest.Query())
	return path
}
