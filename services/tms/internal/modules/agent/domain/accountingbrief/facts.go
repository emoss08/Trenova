package accountingbrief

import (
	"math"
	"slices"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	PastDueThresholdDays      = 90
	CrossingWindowDays        = 7
	DueSoonDays               = 7
	DueThisMonthDays          = 30
	CollectionWindowDays      = 91
	PromiseLookaheadDays      = 2
	UncontactedDays           = 14
	ShortPayRemainderMaxMinor = 5_000
	ConcentrationSharePercent = 30
	DSOMoveTenths             = 30
	maxCustomerName           = 255
)

type Link string

const (
	LinkOpenInvoices       = Link("OpenInvoices")
	LinkOverdue            = Link("Overdue")
	LinkPastNinety         = Link("PastNinety")
	LinkDisputed           = Link("Disputed")
	LinkUnappliedCash      = Link("UnappliedCash")
	LinkTopCustomer        = Link("TopCustomer")
	LinkCrossingNinety     = Link("CrossingNinety")
	LinkDueSoon            = Link("DueSoon")
	LinkBrokenPromise      = Link("BrokenPromise")
	LinkPromisesDue        = Link("PromisesDue")
	LinkUncontacted        = Link("Uncontacted")
	LinkFollowUpsDue       = Link("FollowUpsDue")
	LinkShortPayRemainders = Link("ShortPayRemainders")
	LinkDrafts             = Link("Drafts")
	LinkNoCollections      = Link("NoCollections")
	LinkDueThisMonth       = Link("DueThisMonth")
)

func Links() []Link {
	return []Link{
		LinkOpenInvoices,
		LinkOverdue,
		LinkPastNinety,
		LinkDisputed,
		LinkUnappliedCash,
		LinkTopCustomer,
		LinkCrossingNinety,
		LinkDueSoon,
		LinkBrokenPromise,
		LinkPromisesDue,
		LinkUncontacted,
		LinkFollowUpsDue,
		LinkShortPayRemainders,
		LinkDrafts,
		LinkNoCollections,
		LinkDueThisMonth,
	}
}

func (l Link) IsValid() bool {
	return slices.Contains(Links(), l)
}

func (l Link) String() string { return string(l) }

type Facts struct {
	OpenMinor           int64 `json:"openMinor"`
	OpenInvoices        int   `json:"openInvoices"`
	OverdueMinor        int64 `json:"overdueMinor"`
	OverdueInvoices     int   `json:"overdueInvoices"`
	PastNinetyMinor     int64 `json:"pastNinetyMinor"`
	DisputedMinor       int64 `json:"disputedMinor"`
	DisputedInvoices    int   `json:"disputedInvoices"`
	UnappliedCashMinor  int64 `json:"unappliedCashMinor"`
	UnappliedPayments   int   `json:"unappliedPayments"`
	CollectedTodayMinor int64 `json:"collectedTodayMinor"`
	PaymentsToday       int   `json:"paymentsToday"`
	DSOTenths           int64 `json:"dsoTenths"`
	DSODeltaTenths      int64 `json:"dsoDeltaTenths"`
	CEIPercent          int   `json:"ceiPercent"`

	TopCustomerID           pulid.ID `json:"topCustomerId,omitempty"`
	TopCustomerName         string   `json:"topCustomerName,omitempty"`
	TopCustomerOverdueMinor int64    `json:"topCustomerOverdueMinor"`

	CrossingNinetyInvoices int   `json:"crossingNinetyInvoices"`
	CrossingNinetyMinor    int64 `json:"crossingNinetyMinor"`
	DueSoonInvoices        int   `json:"dueSoonInvoices"`
	DueSoonMinor           int64 `json:"dueSoonMinor"`
	ShortPayRemainders     int   `json:"shortPayRemainders"`
	ShortPayRemainderMinor int64 `json:"shortPayRemainderMinor"`
	DueThisMonthInvoices   int   `json:"dueThisMonthInvoices"`
	DueThisMonthMinor      int64 `json:"dueThisMonthMinor"`
	NextDueInDays          int   `json:"nextDueInDays"`

	DraftInvoices        int   `json:"draftInvoices"`
	DraftMinor           int64 `json:"draftMinor"`
	OldestDraftDays      int   `json:"oldestDraftDays"`
	CollectedWindowMinor int64 `json:"collectedWindowMinor"`
	BilledWindowMinor    int64 `json:"billedWindowMinor"`

	BrokenPromises            int      `json:"brokenPromises"`
	BrokenPromiseMinor        int64    `json:"brokenPromiseMinor"`
	BrokenPromiseCustomerID   pulid.ID `json:"brokenPromiseCustomerId,omitempty"`
	BrokenPromiseCustomerName string   `json:"brokenPromiseCustomerName,omitempty"`
	PromisesDue               int      `json:"promisesDue"`
	PromisesDueMinor          int64    `json:"promisesDueMinor"`
	UncontactedAccounts       int      `json:"uncontactedAccounts"`
	UncontactedOverdueMinor   int64    `json:"uncontactedOverdueMinor"`
	FollowUpsDue              int      `json:"followUpsDue"`

	Timezone string `json:"timezone"`
}

func (f *Facts) normalize() {
	f.TopCustomerName = stringutils.Ellipsize(f.TopCustomerName, maxCustomerName)
	f.BrokenPromiseCustomerName = stringutils.Ellipsize(f.BrokenPromiseCustomerName, maxCustomerName)
}

func (f Facts) Same(other Facts) bool {
	other.Timezone = f.Timezone
	return f == other
}

func (f Facts) HasTopCustomer() bool {
	return !f.TopCustomerID.IsNil() && f.TopCustomerName != "" && f.TopCustomerOverdueMinor > 0
}

func (f Facts) HasBrokenPromiseCustomer() bool {
	return f.BrokenPromises > 0 && !f.BrokenPromiseCustomerID.IsNil() &&
		f.BrokenPromiseCustomerName != ""
}

func (f Facts) HasReceivables() bool {
	return f.OpenMinor != 0 || f.OpenInvoices > 0 || f.UnappliedCashMinor != 0 ||
		f.CollectedTodayMinor != 0
}

func (f Facts) HasDSOBaseline() bool {
	return f.DSOTenths > 0 && f.DSOTenths-f.DSODeltaTenths > 0
}

func (f Facts) DSOMoved() bool {
	return f.HasDSOBaseline() && (f.DSODeltaTenths >= DSOMoveTenths || f.DSODeltaTenths <= -DSOMoveTenths)
}

func (f Facts) NothingCollected() bool {
	return f.BilledWindowMinor > 0 && f.CollectedWindowMinor <= 0 && f.CollectedTodayMinor <= 0
}

func (f Facts) DueThisMonthWorthNaming() bool {
	return f.DueSoonInvoices == 0 && f.DueThisMonthInvoices > 0
}

func (f Facts) TopCustomerSharePercent() int {
	if !f.HasTopCustomer() || f.OverdueMinor <= 0 {
		return 0
	}
	share := float64(f.TopCustomerOverdueMinor) * 100 / float64(f.OverdueMinor)
	return int(math.Round(min(share, 100)))
}

func (f Facts) Concentrated() bool {
	return f.TopCustomerSharePercent() >= ConcentrationSharePercent
}

func (f Facts) Links() []Link {
	present := map[Link]bool{
		LinkOpenInvoices:       f.OpenInvoices > 0,
		LinkOverdue:            f.OverdueInvoices > 0,
		LinkPastNinety:         f.PastNinetyMinor > 0,
		LinkDisputed:           f.DisputedInvoices > 0,
		LinkUnappliedCash:      f.UnappliedCashMinor > 0,
		LinkTopCustomer:        f.HasTopCustomer(),
		LinkCrossingNinety:     f.CrossingNinetyInvoices > 0,
		LinkDueSoon:            f.DueSoonInvoices > 0,
		LinkBrokenPromise:      f.BrokenPromises > 0,
		LinkPromisesDue:        f.PromisesDue > 0,
		LinkUncontacted:        f.UncontactedAccounts > 0,
		LinkFollowUpsDue:       f.FollowUpsDue > 0,
		LinkShortPayRemainders: f.ShortPayRemainders > 0,
		LinkDrafts:             f.DraftInvoices > 0,
		LinkNoCollections:      f.NothingCollected(),
		LinkDueThisMonth:       f.DueThisMonthWorthNaming(),
	}

	out := make([]Link, 0, len(present))
	for _, link := range Links() {
		if present[link] {
			out = append(out, link)
		}
	}
	return out
}

func (f Facts) CustomerFor(link Link) (pulid.ID, string, bool) {
	switch link {
	case LinkTopCustomer:
		if f.HasTopCustomer() {
			return f.TopCustomerID, f.TopCustomerName, true
		}
	case LinkBrokenPromise:
		if f.BrokenPromises == 1 && f.HasBrokenPromiseCustomer() {
			return f.BrokenPromiseCustomerID, f.BrokenPromiseCustomerName, true
		}
	}
	return pulid.Nil, "", false
}

type Action struct {
	Link         Link     `json:"link"`
	Count        int      `json:"count"`
	AmountMinor  int64    `json:"amountMinor"`
	CustomerID   pulid.ID `json:"customerId,omitempty"`
	CustomerName string   `json:"customerName,omitempty"`
}

func (f Facts) Actions() []Action {
	out := make([]Action, 0, 8)
	add := func(link Link, count int, amount int64) {
		action := Action{Link: link, Count: count, AmountMinor: amount}
		if id, name, ok := f.CustomerFor(link); ok {
			action.CustomerID, action.CustomerName = id, name
		}
		out = append(out, action)
	}

	if f.BrokenPromises > 0 {
		add(LinkBrokenPromise, f.BrokenPromises, f.BrokenPromiseMinor)
	}
	if f.CrossingNinetyInvoices > 0 {
		add(LinkCrossingNinety, f.CrossingNinetyInvoices, f.CrossingNinetyMinor)
	}
	if f.UncontactedAccounts > 0 {
		add(LinkUncontacted, f.UncontactedAccounts, f.UncontactedOverdueMinor)
	}
	if f.DraftInvoices > 0 {
		add(LinkDrafts, f.DraftInvoices, f.DraftMinor)
	}
	if f.NothingCollected() {
		add(LinkNoCollections, 0, f.BilledWindowMinor)
	}
	if f.FollowUpsDue > 0 {
		add(LinkFollowUpsDue, f.FollowUpsDue, 0)
	}
	if f.PromisesDue > 0 {
		add(LinkPromisesDue, f.PromisesDue, f.PromisesDueMinor)
	}
	if f.DueSoonInvoices > 0 {
		add(LinkDueSoon, f.DueSoonInvoices, f.DueSoonMinor)
	}
	if f.DueThisMonthWorthNaming() {
		add(LinkDueThisMonth, f.DueThisMonthInvoices, f.DueThisMonthMinor)
	}
	if f.ShortPayRemainders > 0 {
		add(LinkShortPayRemainders, f.ShortPayRemainders, f.ShortPayRemainderMinor)
	}
	if f.UnappliedCashMinor > 0 {
		add(LinkUnappliedCash, f.UnappliedPayments, f.UnappliedCashMinor)
	}
	return out
}
