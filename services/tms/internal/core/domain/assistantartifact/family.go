package assistantartifact

// Family is how the Desk groups the kinds it draws: one mark, one colour
// and one filter per family. A composed view is a table_view that opens
// rather than a table read once, so a family is decided by the payload as
// well as the kind.
type Family string

const (
	FamilyTable    Family = "table"
	FamilyRecord   Family = "record"
	FamilyRate     Family = "rate"
	FamilyEmail    Family = "email"
	FamilyPlan     Family = "plan"
	FamilyReport   Family = "report"
	FamilyDiff     Family = "diff"
	FamilyDoc      Family = "doc"
	FamilyView     Family = "view"
	FamilyDecision Family = "decision"
	FamilyExtract  Family = "extract"
)

// AllFamilies is every family in the order the browser lists its filters.
func AllFamilies() []Family {
	return []Family{
		FamilyTable, FamilyRecord, FamilyRate, FamilyEmail, FamilyPlan, FamilyReport,
		FamilyDiff, FamilyDoc, FamilyView, FamilyDecision, FamilyExtract,
	}
}

func (f Family) IsValid() bool {
	for _, family := range AllFamilies() {
		if f == family {
			return true
		}
	}

	return false
}

// familyOfKind is the family each kind belongs to, except a table_view,
// which is a view when its payload names a page to open.
var familyOfKind = map[Kind]Family{
	KindTableView:       FamilyTable,
	KindReportPreview:   FamilyReport,
	KindReportRun:       FamilyReport,
	KindEntityCard:      FamilyRecord,
	KindRateExplanation: FamilyRate,
	KindEmailDraft:      FamilyEmail,
	KindInboundMessage:  FamilyEmail,
	KindPlan:            FamilyPlan,
	KindRunDiff:         FamilyDiff,
	KindNavigation:      FamilyView,
	KindDashboardRef:    FamilyView,
	KindDecisionRequest: FamilyDecision,
	KindDocument:        FamilyDoc,
	KindBriefing:        FamilyDoc,
	KindDraftEdit:       FamilyDoc,
	KindExtraction:      FamilyExtract,
}

// FamilyOf is the family an artifact is listed under.
func FamilyOf(kind Kind, payload map[string]any) Family {
	if kind == KindTableView {
		if _, opens := payload["path"]; opens {
			return FamilyView
		}
	}
	if family, ok := familyOfKind[kind]; ok {
		return family
	}

	return FamilyDoc
}

// FamilyKinds is the kinds a family covers, and whether a table_view in it
// must name a page (true), must not (false) or either (nil). The repository
// filters on it so the server's families match the ones the Desk draws.
func FamilyKinds(family Family) (kinds []Kind, viewPath *bool) {
	for _, kind := range AllKinds() {
		if familyOfKind[kind] == family || (kind == KindTableView && family == FamilyView) {
			kinds = append(kinds, kind)
		}
	}
	switch family {
	case FamilyTable:
		opens := false
		viewPath = &opens
	case FamilyView:
		opens := true
		viewPath = &opens
	default:
	}

	return kinds, viewPath
}
